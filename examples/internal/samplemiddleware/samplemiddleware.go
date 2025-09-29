package samplemiddleware

import (
	"context"
	"fmt"
	"github.com/jaredmtdev/gather"
	"sync/atomic"
	"time"
)

/*
simple examples of middlewares that can be built to work with this library
*/

// RetryAfter - retries after given time delay
// by default, will always retry unless shouldRetry is defined
func RetryAfter[IN, OUT any](d time.Duration, shouldRetry ...func(i IN) (newIn IN, should bool)) gather.Middleware[IN, OUT] {
	if len(shouldRetry) == 0 {
		shouldRetry = append(shouldRetry, func(i IN) (IN, bool) { return i, true })
	}
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return gather.HandlerFunc[IN, OUT](func(
			ctx context.Context, in IN, s *gather.Scope[IN],
		) (OUT, error) {
			out, err := next(ctx, in, s)
			if err != nil {
				in, should := shouldRetry[0](in)
				if should {
					s.RetryAfter(ctx, in, d)
					return out, err
				}
				return out, fmt.Errorf("max retries hit for %+v. %w", in, err)
			}
			return out, err
		})
	}
}

// Logger - prints input/ouput/error data
func Logger[IN any, OUT any](infoPrefix, errorPrefix string) gather.Middleware[IN, OUT] {
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return gather.HandlerFunc[IN, OUT](func(ctx context.Context, in IN, s *gather.Scope[IN]) (OUT, error) {
			out, err := next(ctx, in, s)
			if err != nil {
				fmt.Println(errorPrefix, in, err)
			} else {
				fmt.Println(infoPrefix, in, out)
			}
			return out, err
		})
	}
}

// Timeout - add timeout on each request
func Timeout[IN any, OUT any](duration time.Duration) gather.Middleware[IN, OUT] {
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return gather.HandlerFunc[IN, OUT](func(ctx context.Context, in IN, s *gather.Scope[IN]) (OUT, error) {
			c, cancel := context.WithTimeout(ctx, duration)
			defer cancel()
			return next(c, in, s)
		})
	}
}

type counter[IN, OUT any] struct {
	every uint64
	count atomic.Uint64
}

// Counter - stateful middleware that keeps track of how many requests are being made
func Counter[IN, OUT any](every int) gather.Middleware[IN, OUT] {
	if every <= 0 {
		every = 1
	}
	c := &counter[IN, OUT]{
		every: uint64(every),
	}
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return gather.HandlerFunc[IN, OUT](func(
			ctx context.Context, in IN, s *gather.Scope[IN],
		) (OUT, error) {
			out, err := next(ctx, in, s)

			n := c.count.Add(1)
			if n%c.every == 0 {
				fmt.Printf("[counter] handled %d requests (every %d)\n", n, c.every)
			}
			return out, err
		})
	}
}
