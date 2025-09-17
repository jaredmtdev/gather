package together_test

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"together"
)

type number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// === generators ===

func gen[T number](ctx context.Context, n T) <-chan T {
	out := make(chan T)
	go func() {
		defer close(out)
		for i := T(0); i < n; i++ {
			select {
			case <-ctx.Done():
				return
			case out <- i:
			}
		}
	}()
	return out
}

// === handlers ===

func convert[FROM number, TO number]() together.HandlerFunc[FROM, TO] {
	return func(_ context.Context, in FROM, _ *together.Scope[FROM]) (TO, error) {
		return TO(in), nil
	}
}

func add[T number](num T) together.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *together.Scope[T]) (T, error) {
		return in + num, nil
	}
}
func subtract[T number](num T) together.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *together.Scope[T]) (T, error) {
		return in - num, nil
	}
}
func multiply[T number](num T) together.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *together.Scope[T]) (T, error) {
		return in * num, nil
	}
}
func divide[T number](num T) together.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *together.Scope[T]) (T, error) {
		return in / num, nil
	}
}

// === middleware ===

func mwDelay[IN, OUT any](d time.Duration) together.Middleware[IN, OUT] {
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *together.Scope[IN]) (OUT, error) {
			select {
			case <-ctx.Done():
				var v OUT
				return v, ctx.Err()
			case <-time.After(d):
			}
			return next(ctx, in, scope)
		}
	}
}

func mwRandomDelay[IN, OUT any](seed int64, minDuration time.Duration, maxDuration time.Duration) together.Middleware[IN, OUT] {
	pool := &sync.Pool{
		New: func() any { return rand.New(rand.NewSource(seed)) },
	}
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *together.Scope[IN]) (OUT, error) {
			r := pool.Get().(*rand.Rand)
			defer pool.Put(r)
			randInterval := time.Duration(r.Int63n(int64(maxDuration - minDuration)))
			delayTime := minDuration + randInterval
			select {
			case <-ctx.Done():
				var v OUT
				return v, ctx.Err()
			case <-time.After(delayTime):
			}
			return next(ctx, in, scope)
		}
	}
}

func mwCancelOnInput[IN comparable, OUT any](cancelOn IN, cancel context.CancelFunc) together.Middleware[IN, OUT] {
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *together.Scope[IN]) (OUT, error) {
			if in == cancelOn {
				cancel()
				var v OUT
				return v, ctx.Err()
			}
			return next(ctx, in, scope)
		}
	}
}

func mwCancelOnCount[IN any, OUT any](cancelOn int32, cancel context.CancelFunc) together.Middleware[IN, OUT] {
	count := atomic.Int32{}
	return func(next together.HandlerFunc[IN, OUT]) together.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *together.Scope[IN]) (OUT, error) {
			if count.Add(1) == cancelOn {
				cancel()
				var v OUT
				return v, ctx.Err()
			}
			return next(ctx, in, scope)
		}
	}
}
