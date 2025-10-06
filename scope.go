package gather

import (
	"context"
	"sync"
	"time"
)

// Scope - a per-request utility passed to handlers and middleware.
type Scope[IN any] struct {
	reenqueue func(IN)
	willRetry bool
	once      sync.Once
	wgJob     *sync.WaitGroup
}

// RetryAfter - retry the request after "delay" time passes.
// only one retry can be done at a time for each job.
// any extra retries will be ignored.
func (s *Scope[IN]) RetryAfter(ctx context.Context, in IN, delay time.Duration) {
	s.once.Do(func() {
		s.willRetry = true
		s.wgJob.Go(func() {
			select {
			case <-ctx.Done():
				s.wgJob.Done()
			case <-time.After(delay):
				s.reenqueue(in)
			}
		})
	})
}

// Retry - will retry immediately.
func (s *Scope[IN]) Retry(ctx context.Context, in IN) {
	s.RetryAfter(ctx, in, 0)
}
