package together

import (
	"context"
	"fmt"
	"sync"

	"together/pkg/syncvalue"
)

// HandlerFunc - function used to handle a single request sent to the worker
// error handling done here. user can:
//   - cancel the context if needed for immediate shutdown
//   - for graceful shutdown: user controls generator. can just close in chan and then let all downstream stages finish
//   - send to their own err channel (which could be processed by another Workers)
//   - use workerHandler for retries, ReplyTo pattern, etc
type HandlerFunc[IN any, OUT any] func(ctx context.Context, in IN, scope *Scope[IN]) (OUT, error)

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
	skip  bool
}

// Pump input data into queue for workers to process
// the workers may also send data back to queue if a retry is triggered.
func (ws *workerStation[IN, OUT]) Pump(ctx context.Context, in <-chan IN) {
	wgPump := sync.WaitGroup{}
	wgPump.Go(func() {
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
		wgPump.Wait()
		ws.wgJob.Wait()
		close(ws.queue)
	}()
}

func (ws *workerStation[IN, OUT]) enqueueFunc(ctx context.Context, index uint64) func(IN) {
	return func(v IN) {
		select {
		case <-ctx.Done():
		case ws.queue <- job[IN]{val: v, index: index}:
		}
	}
}

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
			enqueue:     ws.enqueueFunc(ctx, jobIn.index),
			wgJob:       &ws.wgJob,
			once:        &sync.Once{},
			retryClosed: &syncvalue.Value[bool]{},
		}

		res, err := ws.handler(ctx, jobIn.val, &scope)
		jobOut := job[OUT]{val: res, index: jobIn.index}
		if !scope.willRetry {
			ws.wgJob.Done()
		}
		if ws.orderPreserved && !scope.willRetry {
			if err != nil {
				jobOut.skip = true
			}
			select {
			case <-ctx.Done():
				continue
			case ws.ordered <- jobOut:
			}
		} else if !ws.orderPreserved && !scope.willRetry && err == nil {
			select {
			case <-ctx.Done():
				continue
			case ws.out <- res:
			}
		}
	}
}

func (ws *workerStation[IN, OUT]) Reorder(ctx context.Context) {
	var nextJobOut uint64
	jobOutMap := map[uint64]job[OUT]{}
	for jobOut := range ws.ordered {
		select {
		case <-ctx.Done():
			return
		default:
		}
		jobOutMap[jobOut.index] = jobOut
		for v, ok := jobOutMap[nextJobOut]; ok; v, ok = jobOutMap[nextJobOut] {
			delete(jobOutMap, v.index)
			nextJobOut++
			if v.skip {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case ws.out <- v.val:
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
	ws.ordered = make(chan job[OUT], ws.bufferSize)
	ws.out = make(chan OUT, ws.bufferSize)

	// using internal queue to allow retries to send back to queue
	// separate channel needed because this block doesn't control closing of the in chan
	ws.Pump(ctx, in)

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
		close(ws.ordered)
		wgOrdered.Wait()
		close(ws.out)
	}()
	return ws.out
}
