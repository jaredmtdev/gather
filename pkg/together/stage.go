package together

import (
	"context"
	"sync"
	"time"
)

// Scope - gives the user some ability to do things that require internal mechanisms
type Scope[IN any] struct {
	enqueue   func(v IN)
	willRetry bool
	once      *sync.Once
	wgJob     *sync.WaitGroup
}

func (s *Scope[IN]) RetryAfter(ctx context.Context, in IN, after time.Duration) {
	s.once.Do(func() {
		s.willRetry = true
		s.wgJob.Go(func() {
			select {
			case <-ctx.Done():
				s.wgJob.Done()
			case <-time.After(after):
				s.enqueue(in)
			}
		})
	})
}

// Retry - will retry immediately
func (s *Scope[IN]) Retry(ctx context.Context, in IN) {
	s.RetryAfter(ctx, in, 0)
}

// allow your work to safely spin up a new go routine (in addition to the worker go routine)
// the worker will stay alive until this go routine completes
func (s *Scope[IN]) Go(f func()) {
	s.wgJob.Go(f)
}

// Handler - function used to handle a single request sent to the worker
// error handling done here. user can:
//   - cancel the context if needed for immediate shutdown
//   - for graceful shutdown: user controls generator. can just close in chan and then let all downstream stages finish
//   - send to their own err channel (which could be processed by another Workers)
//   - use workerHandler for retries, ReplyTo pattern, etc
type Handler[IN any, OUT any] func(ctx context.Context, in IN, scope *Scope[IN]) (OUT, error)

// workerStation - configures behavior of Workers
type workerStation struct {
	workerSize int
	bufferSize int
}

type Opt func(w *workerStation)

func WithWorkerSize(workerSize int) Opt {
	return func(w *workerStation) {
		w.workerSize = workerSize
	}
}

func WithBufferSize(bufferSize int) Opt {
	return func(w *workerStation) {
		w.bufferSize = bufferSize
	}
}

func newWorkerStation() *workerStation {
	return &workerStation{
		workerSize: 1,
	}
}

// Workers - build a single pipeline stage based on the handler and options
// error handling is done within work func. in there the user can:
func Workers[IN any, OUT any](ctx context.Context, in <-chan IN, handler Handler[IN, OUT], opts ...Opt) <-chan OUT {
	ws := newWorkerStation()
	for _, opt := range opts {
		opt(ws)
	}

	out := make(chan OUT, ws.bufferSize)

	// using internal queue to allow retries to send back to queue
	// since we don't control closing of in chan
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
					// collect any remaining jobs in queue to zero out the wait group
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
