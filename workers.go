package together

import (
	"context"
	"fmt"
	"sync"
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
	workerSize int
	bufferSize int
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

func newWorkerStation(opts []Opt) *workerStation {
	ws := &workerStation{
		workerSize: 1,
	}
	for _, opt := range opts {
		opt(ws)
	}
	return ws
}

// Workers - build a single pipeline stage based on the handler and options.
func Workers[IN any, OUT any](ctx context.Context, in <-chan IN, handler HandlerFunc[IN, OUT], opts ...Opt) <-chan OUT {
	ws := newWorkerStation(opts)

	out := make(chan OUT, ws.bufferSize)

	// using internal queue to allow retries to send back to queue
	// since this block doesn't control closing of the in chan
	queue := make(chan IN, ws.bufferSize)
	wgPump := sync.WaitGroup{}
	wgPump.Add(1)
	wgJob := sync.WaitGroup{}
	go func() {
		defer wgPump.Done()
		for v := range in {
			wgJob.Add(1)
			select {
			case <-ctx.Done():
				wgJob.Done()
				return
			case queue <- v:
			}
		}
	}()

	go func() {
		wgPump.Wait()
		wgJob.Wait()
		close(queue)
	}()

	enqueue := func(v IN) {
		select {
		case <-ctx.Done():
		case queue <- v:
		}
	}

	wgWorker := sync.WaitGroup{}
	wgWorker.Add(ws.workerSize)
	for range ws.workerSize {
		go func() {
			defer wgWorker.Done()
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
					enqueue: enqueue,
					wgJob:   &wgJob,
					once:    &sync.Once{},
				}

				res, err := handler(ctx, v, &scope)
				if !scope.willRetry {
					wgJob.Done()
				}
				if err == nil {
					select {
					case <-ctx.Done():
						return
					case out <- res:
					}
				}
			}
		}()
	}
	go func() {
		wgWorker.Wait()
		close(out)
	}()
	return out
}
