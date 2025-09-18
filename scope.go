package together

import (
	"context"
	"sync"
	"time"

	"together/pkg/syncvalue"
)

// Scope - gives the user some ability to do things that require internal mechanisms.
type Scope[IN any] struct {
	enqueue     func(IN)
	willRetry   bool
	retryClosed *syncvalue.Value[bool]
	once        *sync.Once
	wgJob       *sync.WaitGroup
}

// RetryAfter - retry the request after "after" time passes.
func (s *Scope[IN]) RetryAfter(ctx context.Context, in IN, after time.Duration) {
	if s.retryClosed.Load() {
		panic("Invalid attempt to retry. Retries can only be executed BEFORE spawning new go routines.")
	}
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

// Retry - will retry immediately.
func (s *Scope[IN]) Retry(ctx context.Context, in IN) {
	s.RetryAfter(ctx, in, 0)
}

// Go - allow your handler to safely spin up a new go routine (in addition to the worker go routine).
// the worker will stay alive until this go routine completes.
func (s *Scope[IN]) Go(f func()) {
	s.retryClosed.Store(true)
	s.wgJob.Go(f)
}
