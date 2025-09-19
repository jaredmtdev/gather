package together

import (
	"context"
	"fmt"
	"sync"

	"together/pkg/syncvalue"
)

// workerOpts - configures behavior of Workers.
type workerOpts struct {
	workerSize     int
	bufferSize     int
	orderPreserved bool
}

// Opt - options used to configure Workers.
type Opt func(w *workerOpts)

// WithWorkerSize - set number of concurrent workers.
func WithWorkerSize(workerSize int) Opt {
	if workerSize <= 0 {
		panic(fmt.Sprintf("must use at least 1 worker! workerSize: %v", workerSize))
	}
	return func(w *workerOpts) {
		w.workerSize = workerSize
	}
}

// WithBufferSize - set buffer size for the internal and output channel.
func WithBufferSize(bufferSize int) Opt {
	if bufferSize < 0 {
		panic(fmt.Sprintf("buffer must be at least 0! bufferSize: %v", bufferSize))
	}
	return func(w *workerOpts) {
		w.bufferSize = bufferSize
	}
}

// WithOrderPreserved - preserves order of input to output
// the workers will keep running but results are blocked from sending until the "next" result is ready to send.
func WithOrderPreserved() Opt {
	return func(w *workerOpts) {
		w.orderPreserved = true
	}
}

func newWorkerOpts(opts []Opt) *workerOpts {
	wo := &workerOpts{
		workerSize: 1,
	}
	for _, opt := range opts {
		opt(wo)
	}
	return wo
}

// workerStation - internal functionality used by Workers.
type workerStation[IN, OUT any] struct {
	*workerOpts

	queue   chan job[IN]
	ordered chan job[OUT]
	out     chan OUT
	wgJob   sync.WaitGroup
	handler HandlerFunc[IN, OUT]
}

// job - keeps track of index for optional ordered results.
type job[T any] struct {
	index uint64
	val   T
	err   error
}

// Enqueue - enqueues input data for workers to process.
// this "middleman" logic is used to allow retries to send jobs back into queue
// note that we can't send to in chan because we don't control when in chan is closed.
func (ws *workerStation[IN, OUT]) Enqueue(ctx context.Context, in <-chan IN) {
	wgEnqueue := sync.WaitGroup{}
	wgEnqueue.Go(func() {
		var indexCounter uint64
		for v := range in {
			ws.wgJob.Add(1)
			select {
			case <-ctx.Done():
				ws.wgJob.Done()
				return
			case ws.queue <- job[IN]{val: v, index: indexCounter}:
				indexCounter++
			}
		}
	})

	go func() {
		wgEnqueue.Wait()
		ws.wgJob.Wait()
		close(ws.queue)
	}()
}

// buildEnqueueFunc - builds the enqueue func used during retries to send (possibly  modified) input back into queue.
func (ws *workerStation[IN, OUT]) buildEnqueueFunc(ctx context.Context, index uint64) func(IN) {
	return func(v IN) {
		select {
		case <-ctx.Done():
		case ws.queue <- job[IN]{val: v, index: index}:
		}
	}
}

// SendResult - sends result from worker to the next step (either out or reorder).
func (ws *workerStation[IN, OUT]) SendResult(ctx context.Context, jobOut job[OUT], err error) {
	defer ws.wgJob.Done()
	if ws.orderPreserved {
		select {
		case <-ctx.Done():
			return
		case ws.ordered <- jobOut:
		}
		return
	}
	if err == nil {
		select {
		case <-ctx.Done():
			return
		case ws.out <- jobOut.val:
		}
	}
}

// StartWorker - starts a single worker to ingest the queue.
func (ws *workerStation[IN, OUT]) StartWorker(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}

	for jobIn := range ws.queue {
		select {
		case <-ctx.Done():
			ws.wgJob.Done()
			// drain any remaining jobs in queue to zero out the wait group
			continue
		default:
		}
		scope := Scope[IN]{
			enqueue:     ws.buildEnqueueFunc(ctx, jobIn.index),
			wgJob:       &ws.wgJob,
			once:        &sync.Once{},
			retryClosed: &syncvalue.Value[bool]{},
		}

		res, err := ws.handler(ctx, jobIn.val, &scope)
		jobOut := job[OUT]{val: res, err: err, index: jobIn.index}
		if !scope.willRetry {
			ws.SendResult(ctx, jobOut, err)
		}
	}
}

// Reorder - used to cache the result until the "next" result is cached and ready to be sent to out chan
// in other words: reorder to make sure all results are sent in the same order of their inputs.
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

// Workers - build a single pipeline stage based on the handler and options.
func Workers[IN any, OUT any](ctx context.Context, in <-chan IN, handler HandlerFunc[IN, OUT], opts ...Opt) <-chan OUT {
	ws := &workerStation[IN, OUT]{
		workerOpts: newWorkerOpts(opts),
		handler:    handler,
	}
	ws.queue = make(chan job[IN], ws.bufferSize)
	if ws.orderPreserved {
		ws.ordered = make(chan job[OUT], ws.bufferSize)
	}
	ws.out = make(chan OUT, ws.bufferSize)

	ws.Enqueue(ctx, in)

	wgWorker := sync.WaitGroup{}
	for range ws.workerSize {
		wgWorker.Go(func() {
			ws.StartWorker(ctx)
		})
	}

	wgOrdered := sync.WaitGroup{}
	if ws.orderPreserved {
		wgOrdered.Go(func() {
			ws.Reorder(ctx)
		})
	}

	go func() {
		wgWorker.Wait()
		if ws.orderPreserved {
			close(ws.ordered)
			wgOrdered.Wait()
		}
		close(ws.out)
	}()
	return ws.out
}
