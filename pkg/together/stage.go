package together

import (
	"context"
	"sync"
	"time"
)

/*
the goal is to offload as much of the complexities as possible to middleware
keep Workers as simple/intuitive as possible

keep everything in one package for now. 3 main components:
-Workers
-Middleware
-Pipeline


what if the user wants an error channel + return type? what if they want a single return type that includes both?
-return type is always provided. for error channel: user controls all generators. they can use "work" func to send err to errCh

what if user wants to stop on first error like errgroup?
they can cancel context on first error and handle error which ever way they want (send to err channel or handle right here)

what if they want to collect all errors and results?
user can send errors to err channel. a separate worker can consume them

what if user wants retries? (probably want to track number of retries for each data)
-use WorkHandler for this (used within "work")

the middleware approach does in fact seem like it can help for just about any use case

time out option on each stage???
-no: this can be done via middleware

*/

// this handler is to give the user the ability to do things that require internal mechanisms
type Scope[IN any] struct {
	enqueue func(v IN)
	wg      *sync.WaitGroup
	once    *sync.Once
}

func (s *Scope[IN]) RetryAfter(in IN, after time.Duration) {
	s.once.Do(func() {
		s.wg.Add(1)
		time.AfterFunc(after, func() {
			defer s.wg.Done()
			s.enqueue(in)
		})
	})
}

func (s *Scope[IN]) Retry(in IN) {
	s.RetryAfter(in, 0)
}

// allow your work to safely spin up a new go routine
// note that the "work" is executed in a worker which is already running on it's own go routine
//
// this might be useful for patterns like ReplyTo
func (s *Scope[IN]) Go(f func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		f()
	}()
}

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

// func WithMiddleware[IN any, OUT any](middleware Middleware[IN, OUT]) Opt {
// 	return func(w *workerStation[IN, OUT]) {
// 		w.middleware = middleware
// 	}
// }

func newWorkerStation() *workerStation {
	return &workerStation{
		workerSize: 1,
	}
}

// error handling is done within work func. in there the user can:
func Workers[IN any, OUT any](ctx context.Context, in <-chan IN, handler Handler[IN, OUT], opts ...Opt) <-chan OUT {
	ws := newWorkerStation()
	for _, opt := range opts {
		opt(ws)
	}

	out := make(chan OUT, ws.bufferSize)

	// using internal queue to allow retries
	queue := make(chan IN, ws.bufferSize)
	wgQueue := sync.WaitGroup{}
	wgQueue.Add(1)
	go func() {
		defer wgQueue.Done()
		for v := range in {
			select {
			case <-ctx.Done():
				return
			case queue <- v:
			}
		}
	}()

	enqueue := func(v IN) {
		select {
		case <-ctx.Done():
		case queue <- v:
		}
	}

	wg := sync.WaitGroup{}
	wg.Add(ws.workerSize)
	for range ws.workerSize {
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				return
			default:
			}

			for v := range queue {
				scope := Scope[IN]{
					enqueue: enqueue,
					wg:      &wgQueue,
					once:    &sync.Once{},
				}
				res, err := handler(ctx, v, &scope)
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
		wgQueue.Wait()
		close(queue)
		wg.Wait()
		close(out)
	}()
	return out
}
