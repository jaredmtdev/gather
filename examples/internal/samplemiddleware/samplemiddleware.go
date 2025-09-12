package samplemiddleware

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"
	"together/pkg/together"
)

/*
simplified versions of middlewares you can build to work with this library
*/

func RetryAfter[IN, OUT any](d time.Duration) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return together.Handler[IN, OUT](func(
			ctx context.Context, in IN, s *together.Scope[IN],
		) (OUT, error) {
			out, err := next(ctx, in, s)
			if err != nil {
				s.RetryAfter(ctx, in, d)
			}
			return out, err
		})
	}
}

func Logger[IN any, OUT any](infoPrefix, errorPrefix string) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return together.Handler[IN, OUT](func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
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

func Timeout[IN any, OUT any](duration time.Duration) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return together.Handler[IN, OUT](func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
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

func Counter[IN, OUT any](every int) together.Middleware[IN, OUT] {
	if every <= 0 {
		every = 1
	}
	c := &counter[IN, OUT]{
		every: uint64(every),
	}
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return together.Handler[IN, OUT](func(
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
