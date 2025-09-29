package gather_test

import (
	"context"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaredmtdev/gather"
)

type number interface {
	~int | ~int32 | ~int64 | ~float32 | ~float64
}

// === generators ===

func gen[T number](ctx context.Context, n T, buffer ...int) <-chan T {
	var bufferSize int
	if len(buffer) > 0 && buffer[0] > 0 {
		bufferSize = buffer[0]
	}
	out := make(chan T, bufferSize)
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

func convert[FROM number, TO number]() gather.HandlerFunc[FROM, TO] {
	return func(_ context.Context, in FROM, _ *gather.Scope[FROM]) (TO, error) {
		return TO(in), nil
	}
}

func add[T number](num T) gather.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *gather.Scope[T]) (T, error) {
		return in + num, nil
	}
}
func subtract[T number](num T) gather.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *gather.Scope[T]) (T, error) {
		return in - num, nil
	}
}
func multiply[T number](num T) gather.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *gather.Scope[T]) (T, error) {
		return in * num, nil
	}
}
func divide[T number](num T) gather.HandlerFunc[T, T] {
	return func(_ context.Context, in T, _ *gather.Scope[T]) (T, error) {
		return in / num, nil
	}
}

// === middleware ===

func mwDelay[IN, OUT any](d time.Duration) gather.Middleware[IN, OUT] {
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *gather.Scope[IN]) (OUT, error) {
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

func mwRandomDelay[IN, OUT any](seed int64, minDuration time.Duration, maxDuration time.Duration) gather.Middleware[IN, OUT] {
	pool := &sync.Pool{
		New: func() any { return rand.New(rand.NewSource(seed)) },
	}
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *gather.Scope[IN]) (OUT, error) {
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

func mwCancelOnInput[IN comparable, OUT any](cancelOn IN, cancel context.CancelFunc) gather.Middleware[IN, OUT] {
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *gather.Scope[IN]) (OUT, error) {
			if in == cancelOn {
				cancel()
				var v OUT
				return v, ctx.Err()
			}
			return next(ctx, in, scope)
		}
	}
}

func mwCancelOnCount[IN any, OUT any](cancelOn int32, cancel context.CancelFunc) gather.Middleware[IN, OUT] {
	count := atomic.Int32{}
	return func(next gather.HandlerFunc[IN, OUT]) gather.HandlerFunc[IN, OUT] {
		return func(ctx context.Context, in IN, scope *gather.Scope[IN]) (OUT, error) {
			if count.Add(1) >= cancelOn {
				cancel()
				var v OUT
				return v, ctx.Err()
			}
			return next(ctx, in, scope)
		}
	}
}
