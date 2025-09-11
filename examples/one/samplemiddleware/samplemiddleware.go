package samplemiddleware

import (
	"context"
	"fmt"
	"time"
	"together/pkg/together"
)

/*
simplified versions of middlewares you can build to work with this library
*/

func RetryAfter[IN any, OUT any](duration time.Duration) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
			out, err := next(ctx, in, s)
			if err != nil {
				s.RetryAfter(in, duration)
			}
			return out, err
		}
	}
}

func Logger[IN any, OUT any](infoPrefix, errorPrefix string) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
			out, err := next(ctx, in, s)
			if err != nil {
				fmt.Println(errorPrefix, err)
			} else {
				fmt.Println(infoPrefix, in, out)
			}
			return out, err
		}
	}
}

func Timeout[IN any, OUT any](duration time.Duration) together.Middleware[IN, OUT] {
	return func(next together.Handler[IN, OUT]) together.Handler[IN, OUT] {
		return func(ctx context.Context, in IN, s *together.Scope[IN]) (OUT, error) {
			c, cancel := context.WithTimeout(ctx, duration)
			defer cancel()
			return next(c, in, s)
		}
	}
}
