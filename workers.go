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

// workerStation - configures behavior of Workers.
type workerStation struct {
	workerSize     int
	bufferSize     int
	orderPreserved bool
}

// Opt - options used to configure Workers.
type Opt func(w *workerStation)

// WithWorkerSize - set number of concurrent workers.
func WithWorkerSize(workerSize int) Opt {
	if workerSize <= 0 {
		panic(fmt.Sprintf("must use at least 1 worker! workerSize: %v", workerSize))
	}
	return func(w *workerStation) {
		w.workerSize = workerSize
	}
}

// WithBufferSize - set buffer size for the internal and output channel.
func WithBufferSize(bufferSize int) Opt {
	if bufferSize < 0 {
		panic(fmt.Sprintf("buffer must be at least 0! bufferSize: %v", bufferSize))
	}
	return func(w *workerStation) {
		w.bufferSize = bufferSize
	}
}

// WithOrderPreserved - preserves order of input to output
// the workers will keep running but results are blocked from sending until the "next" result is ready to send.
func WithOrderPreserved() Opt {
	return func(w *workerStation) {
		w.orderPreserved = true
	}
}

func newWorkerStation(opts []Opt) *workerStation {
	ws := &workerStation{
		workerSize: 1,
	}
	for _, opt := range opts {
		opt(ws)
	}
	return ws
}

// job - keeps track of index for optional ordered results.
type job[T any] struct {
	index uint64
	val   T
	skip  bool
}

// pump input data into queue for workers to process
// the workers may also send data back to queue if a retry is triggered.
func pump[IN any](ctx context.Context, ws *workerStation, in <-chan IN) (chan job[IN], *sync.WaitGroup) {
	queue := make(chan job[IN], ws.bufferSize)
	wgJob := sync.WaitGroup{}
	wgPump := sync.WaitGroup{}
	wgPump.Go(func() {
		var indexCounter uint64
		for v := range in {
			wgJob.Add(1)
			select {
			case <-ctx.Done():
				wgJob.Done()
				return
			case queue <- job[IN]{val: v, index: indexCounter}:
				indexCounter++
			}
		}
	})

	go func() {
		wgPump.Wait()
		wgJob.Wait()
		close(queue)
	}()
	return queue, &wgJob
}

// Workers - build a single pipeline stage based on the handler and options.
func Workers[IN any, OUT any](ctx context.Context, in <-chan IN, handler HandlerFunc[IN, OUT], opts ...Opt) <-chan OUT {
	ws := newWorkerStation(opts)

	ordered := make(chan job[OUT], ws.bufferSize)
	out := make(chan OUT, ws.bufferSize)

	// using internal queue to allow retries to send back to queue
	// separate channel needed because this block doesn't control closing of the in chan
	queue, wgJob := pump(ctx, ws, in)

	enqueue := func(index uint64) func(IN) {
		return func(v IN) {
			select {
			case <-ctx.Done():
			case queue <- job[IN]{val: v, index: index}:
			}
		}
	}

	wgWorker := sync.WaitGroup{}
	for range ws.workerSize {
		wgWorker.Go(func() {
			select {
			case <-ctx.Done():
				return
			default:
			}

			for v := range queue {
				select {
				case <-ctx.Done():
					wgJob.Done()
					// drain any remaining jobs in queue to zero out the wait group
					continue
				default:
				}
				scope := Scope[IN]{
					enqueue:     enqueue(v.index),
					wgJob:       wgJob,
					once:        &sync.Once{},
					retryClosed: &syncvalue.Value[bool]{},
				}

				res, err := handler(ctx, v.val, &scope)
				jobOut := job[OUT]{val: res, index: v.index}
				if !scope.willRetry {
					wgJob.Done()
				}
				if ws.orderPreserved && !scope.willRetry {
					if err != nil {
						jobOut.skip = true
					}
					select {
					case <-ctx.Done():
						continue
					case ordered <- jobOut:
					}
				} else if !ws.orderPreserved && !scope.willRetry && err == nil {
					select {
					case <-ctx.Done():
						continue
					case out <- res:
					}
				}
			}
		})
	}

	wgOrdered := sync.WaitGroup{}
	wgOrdered.Go(func() {
		var nextJobOut uint64
		jobOutMap := map[uint64]job[OUT]{}
		for jobOut := range ordered {
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
				case out <- v.val:
				}
			}
		}
	})

	go func() {
		wgWorker.Wait()
		close(ordered)
		wgOrdered.Wait()
		close(out)
	}()
	return out
}
