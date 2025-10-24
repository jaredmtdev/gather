package gather

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// workerOpts - configures behavior of Workers.
type workerOpts struct {
	maxWorkerSize  int64
	minWorkerSize  int64
	ttlElastic     time.Duration
	elasticWorkers bool
	bufferSize     int
	orderPreserved bool
	panicOnNil     bool
}

func (wo *workerOpts) Validate() error {
	if wo.maxWorkerSize <= 0 {
		return newInvalidWorkerSizeError(wo.maxWorkerSize)
	}
	if wo.bufferSize < 0 {
		return newInvalidBufferSizeError(wo.bufferSize)
	}
	if wo.minWorkerSize < 0 {
		return newInvalidMinWorkerSizeError(wo.minWorkerSize)
	}
	if wo.ttlElastic < 0 {
		return newInvalidTTLError(wo.ttlElastic)
	}
	if wo.minWorkerSize > wo.maxWorkerSize {
		return newMinWorkerSizeTooLargeError(wo.minWorkerSize, wo.maxWorkerSize)
	}
	return nil
}

// Opt - options used to configure Workers.
type Opt func(w *workerOpts)

// WithWorkerSize - set the number of concurrent workers.
// Each worker consumes one job at a time.
//
// Uses 1 by default.
func WithWorkerSize(workerSize int) Opt {
	return func(w *workerOpts) {
		w.maxWorkerSize = int64(workerSize)
	}
}

// WithBufferSize - set buffer size for the internal and output channels.
//
// Uses unbuffered channels by default.
func WithBufferSize(bufferSize int) Opt {
	return func(w *workerOpts) {
		w.bufferSize = bufferSize
	}
}

// WithOrderPreserved - preserves order of input to output.
// The workers will keep running but results are blocked from sending to out until the "next" result is ready to send.
func WithOrderPreserved() Opt {
	return func(w *workerOpts) {
		w.orderPreserved = true
	}
}

// WithPanicOnNilChannel - option to panic when a nil channel is sent to Workers.
// By default, Workers will immediately close the out channel and return.
func WithPanicOnNilChannel() Opt {
	return func(w *workerOpts) {
		w.panicOnNil = true
	}
}

// WithElasticWorkers - makes workers elastic: automatically scale up and down as needed.
// Start at `minWorkerSize` workers and scale up to `workerSize`.
// The scaling is done incrementally. Each increment is applied after `ttl`.
func WithElasticWorkers(minWorkerSize int, ttl time.Duration) Opt {
	return func(w *workerOpts) {
		w.minWorkerSize = int64(minWorkerSize)
		w.ttlElastic = ttl
		w.elasticWorkers = true
	}
}

func newWorkerOpts(opts []Opt) *workerOpts {
	wo := &workerOpts{
		maxWorkerSize: 1,
	}
	for _, opt := range opts {
		opt(wo)
	}
	if !wo.elasticWorkers {
		wo.minWorkerSize = wo.maxWorkerSize
	}
	if err := wo.Validate(); err != nil {
		panic(err.Error())
	}
	return wo
}

// workerStation - provides context for Workers.
type workerStation[IN, OUT any] struct {
	*workerOpts

	queue       chan job[IN]
	ordered     chan job[OUT]
	out         chan OUT
	wgJob       sync.WaitGroup
	wgWorker    sync.WaitGroup
	handler     HandlerFunc[IN, OUT]
	workerCount atomic.Int64
}

// job - wraps around incoming and outgoing data (val) to track job metadata.
// used primarily for tracking order of each job.
type job[T any] struct {
	index uint64
	val   T
	err   error
}

func (ws *workerStation[IN, OUT]) getInput(ctx context.Context, in <-chan IN) (IN, bool) {
	select {
	case v, ok := <-in:
		return v, ok
	case <-ctx.Done():
		var v IN
		return v, false
	}
}

func (ws *workerStation[IN, OUT]) enqueueLoop(ctx context.Context, in <-chan IN) {
	var indexCounter uint64
	for {
		inputValue, ok := ws.getInput(ctx, in)
		if !ok {
			return
		}
		ws.wgJob.Add(1)

		if ws.elasticWorkers {
			select {
			case <-ctx.Done():
				ws.wgJob.Done()
				return
			case ws.queue <- job[IN]{val: inputValue, index: indexCounter}:
				indexCounter++
				continue
			default:
			}
			workerCount := ws.workerCount.Load()
			if workerCount >= ws.minWorkerSize && workerCount < ws.maxWorkerSize {
				ws.AddWorker(ctx)
				ws.workerCount.Add(1)
			}
		}

		select {
		case <-ctx.Done():
			ws.wgJob.Done()
			return
		case ws.queue <- job[IN]{val: inputValue, index: indexCounter}:
			indexCounter++
		}
	}
}

// Enqueue - enqueues input data for workers to process.
// this "middleman" logic is used to allow retries to send jobs back into queue
// note that we can't send to in chan because we don't control when in chan is closed.
func (ws *workerStation[IN, OUT]) Enqueue(ctx context.Context, in <-chan IN) {
	wgEnqueue := sync.WaitGroup{}
	wgEnqueue.Go(func() {
		ws.enqueueLoop(ctx, in)
	})

	go func() {
		wgEnqueue.Wait()
		ws.wgJob.Wait()
		close(ws.queue)
	}()
}

// buildReenqueueFunc - builds the re-enqueue func used during retries to send (possibly modified) input back into queue.
func (ws *workerStation[IN, OUT]) buildReenqueueFunc(ctx context.Context, index uint64) func(IN) {
	return func(v IN) {
		select {
		case <-ctx.Done():
			ws.wgJob.Done()
		case ws.queue <- job[IN]{val: v, index: index}:
		}
	}
}

// SendResult - sends result from worker to the next step (either out or reorder gate).
func (ws *workerStation[IN, OUT]) SendResult(ctx context.Context, jobOut job[OUT], err error) {
	defer ws.wgJob.Done()
	if ws.orderPreserved {
		select {
		case <-ctx.Done():
		case ws.ordered <- jobOut:
		}
		return
	}
	if err == nil {
		select {
		case <-ctx.Done():
		case ws.out <- jobOut.val:
		}
	}
}

func (ws *workerStation[IN, OUT]) wgJobFlush() {
	for range ws.queue {
		ws.wgJob.Done()
	}
}

// StartWorker - starts a single worker to ingest the queue.
func (ws *workerStation[IN, OUT]) StartWorker(ctx context.Context) {
	defer ws.workerCount.Add(-1)
	var tick <-chan time.Time
	for {
		if ws.elasticWorkers {
			tick = time.After(ws.ttlElastic)
		}
		var jobIn job[IN]
		var ok bool
		select {
		case <-ctx.Done():
			ws.wgJobFlush()
			return
		case <-tick:
			// TODO: use a limiter to slow down scaling down
			if ws.workerCount.Load() > ws.minWorkerSize {
				return
			}
			continue
		case jobIn, ok = <-ws.queue:
			if !ok {
				return
			}
		}
		scope := Scope[IN]{
			reenqueue: ws.buildReenqueueFunc(ctx, jobIn.index),
			wgJob:     &ws.wgJob,
		}

		res, err := ws.handler(ctx, jobIn.val, &scope)
		jobOut := job[OUT]{val: res, err: err, index: jobIn.index}
		if !scope.willRetry {
			ws.SendResult(ctx, jobOut, err)
		}
	}
}

func (ws *workerStation[IN, OUT]) AddWorker(ctx context.Context) {
	ws.wgWorker.Go(func() {
		ws.StartWorker(ctx)
	})
}

// Reorder - gate used to cache the result until the "next" result is cached and ready to be sent to out chan.
// makes sure all results are sent to out chan in the same order it was received from in chan.
func (ws *workerStation[IN, OUT]) Reorder(ctx context.Context) {
	var nextJobOutIndex uint64
	jobOutCache := map[uint64]job[OUT]{}
	for jobOutReceived := range ws.ordered {
		select {
		case <-ctx.Done():
			return
		default:
		}
		jobOutCache[jobOutReceived.index] = jobOutReceived
		for jobOut, ok := jobOutCache[nextJobOutIndex]; ok; jobOut, ok = jobOutCache[nextJobOutIndex] {
			delete(jobOutCache, jobOut.index)
			nextJobOutIndex++
			if jobOut.err != nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ws.out <- jobOut.val:
			}
		}
	}
}

// Workers starts multiple goroutines that read jobs from `in`,
// process them with handler, and forward results to `out`. It returns `out`.
//
// Concurrency and resource use:
//   - Spawns N worker goroutines (configured with `opts`)
//     plus a small, constant number of internal coordinators
//     (O(1)). No goroutines are created per job.
//   - Backpressure is applied by the capacities of `in`/`out` (unbuffered channels block).
//   - Buffer size of internal and out channels can be configured with opts
//
// Lifecycle:
//   - When `in` is closed and all jobs are drained/processed, `out` is closed.
//   - If `ctx` is canceled, workers stop early and out is closed after in-flight jobs exit.
//   - The `out` channel MUST be drained to avoid a deadlock.
//
// Ordering:
//   - By default, results are NOT guaranteed to preserve input order.
//   - Use `opts` to configure to guarantee preserved order.
//
// Errors:
//   - If handler returns an error, the job is NOT sent to `out`.
func Workers[IN any, OUT any](
	ctx context.Context,
	in <-chan IN,
	handler HandlerFunc[IN, OUT],
	opts ...Opt,
) <-chan OUT {
	ws := &workerStation[IN, OUT]{
		workerOpts: newWorkerOpts(opts),
		handler:    handler,
	}
	ws.queue = make(chan job[IN], ws.bufferSize)
	if ws.orderPreserved {
		ws.ordered = make(chan job[OUT], ws.bufferSize)
	}
	ws.out = make(chan OUT, ws.bufferSize)

	if in == nil {
		if ws.panicOnNil {
			panic("gather.Workers: nil input channel")
		}
		close(ws.out)
		return ws.out
	}

	ws.Enqueue(ctx, in)

	ws.wgWorker = sync.WaitGroup{}
	for range ws.minWorkerSize {
		ws.AddWorker(ctx)
	}
	ws.workerCount.Add(ws.minWorkerSize)

	wgOrdered := sync.WaitGroup{}
	if ws.orderPreserved {
		wgOrdered.Go(func() {
			ws.Reorder(ctx)
		})
	}

	go func() {
		ws.wgWorker.Wait()
		if ws.orderPreserved {
			close(ws.ordered)
			wgOrdered.Wait()
		}
		close(ws.out)
	}()
	return ws.out
}
