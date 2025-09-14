package samplemiddleware

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
	"together"
)

/*
simplified versions of middlewares you can build to work with this library
*/

// RetryAfter - retries after given time delay
// by default, will always retry unless shouldRetry is defined
func RetryAfter[IN, OUT any](d time.Duration, shouldRetry ...func(i IN) (newIn IN, should bool)) together.Middleware[IN, OUT] {
	if len(shouldRetry) == 0 {
		shouldRetry = append(shouldRetry, func(i IN) (IN, bool) { return i, true })
	}
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return together.HandlerFunc[IN, OUT](func(
			ctx context.Context, in IN, s *together.Scope[IN],
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
func Logger[IN any, OUT any](infoPrefix, errorPrefix string) together.Middleware[IN, OUT] {
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return together.HandlerFunc[IN, OUT](func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
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
func Timeout[IN any, OUT any](duration time.Duration) together.Middleware[IN, OUT] {
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return together.HandlerFunc[IN, OUT](func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
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
func Counter[IN, OUT any](every int) together.Middleware[IN, OUT] {
	if every <= 0 {
		every = 1
	}
	c := &counter[IN, OUT]{
		every: uint64(every),
	}
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return together.HandlerFunc[IN, OUT](func(
			ctx context.Context, in IN, s *together.Scope[IN],
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
