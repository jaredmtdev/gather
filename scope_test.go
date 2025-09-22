package gather_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"gather"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScopeRetry(t *testing.T) {
	ctx := context.Background()
	totalAttempts := atomic.Int32{}

	handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == 2 || in == 3 {
			scope.Retry(ctx, in+1)
			return 0, errors.New("invalid number")
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	assert.Equal(t, int32(23), totalAttempts.Load())
}

func TestScopeRetryAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		totalAttempts := atomic.Int32{}
		delayUntilRetry := 100 * time.Millisecond

		handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
			totalAttempts.Add(1)
			if in == 2 || in == 3 {
				scope.RetryAfter(ctx, in+1, delayUntilRetry)
				return 0, errors.New("invalid number")
			}
			return in * 2, nil
		})

		start := time.Now()
		var got int
		for range gather.Workers(ctx, gen(ctx, 10), handler, gather.WithWorkerSize(10), gather.WithBufferSize(10)) {
			got++
		}
		duration := time.Since(start)
		assert.Equal(t, 10, got)
		assert.Equal(t, int32(13), totalAttempts.Load())
		assert.LessOrEqual(t, delayUntilRetry.Milliseconds(), duration.Milliseconds())
	})
}

func TestScopeRetryAfterWhenNoError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		totalAttempts := atomic.Int32{}
		delayUntilRetry := 100 * time.Millisecond

		handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
			totalAttempts.Add(1)
			if in == 2 || in == 3 {
				scope.RetryAfter(ctx, in+1, delayUntilRetry)
				return 0, nil
			}
			return in * 2, nil
		})

		start := time.Now()
		var got int
		for range gather.Workers(ctx, gen(ctx, 10), handler, gather.WithWorkerSize(10), gather.WithBufferSize(10)) {
			got++
		}
		duration := time.Since(start)
		assert.Equal(t, 10, got)
		assert.Equal(t, int32(13), totalAttempts.Load())
		assert.LessOrEqual(t, delayUntilRetry.Milliseconds(), duration.Milliseconds())
	})
}

func TestScopeRetryAfterOnLastItem(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		totalAttempts := atomic.Int32{}
		delayUntilRetry := 100 * time.Millisecond

		handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
			totalAttempts.Add(1)
			if in == 9 {
				scope.RetryAfter(ctx, in+1, delayUntilRetry)
				return 0, errors.New("invalid number")
			}
			return in * 2, nil
		})

		start := time.Now()
		var got int
		for range gather.Workers(ctx, gen(ctx, 10), handler, gather.WithWorkerSize(2), gather.WithBufferSize(1)) {
			got++
		}
		duration := time.Since(start)
		assert.Equal(t, 10, got)
		assert.Equal(t, int32(11), totalAttempts.Load())
		assert.LessOrEqual(t, delayUntilRetry.Milliseconds(), duration.Milliseconds())
	})
}

func TestScopeWithMultipleRetries(t *testing.T) {
	ctx := context.Background()
	totalAttempts := atomic.Int32{}

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == 2 || in == 3 {
			scope.Retry(ctx, in+1)
			scope.Retry(ctx, in+1)
			scope.Retry(ctx, in+1)
			return 0, errors.New("invalid number")
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	// expect scope.Retry to only be able to execute once per request
	assert.Equal(t, int32(23), totalAttempts.Load())
}

func TestScopeGo(t *testing.T) {
	ctx := context.Background()
	counter := atomic.Int32{}
	handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, scope *gather.Scope[int]) (int, error) {
		if in == 2 || in == 3 {
			scope.Go(
				func() {
					counter.Add(1)
				},
			)
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	assert.Equal(t, int32(2), counter.Load())
}

func TestScopeGoNested(t *testing.T) {
	ctx := context.Background()
	counter := atomic.Int32{}
	handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, scope *gather.Scope[int]) (int, error) {
		if in == 2 || in == 3 {
			scope.Go(
				func() {
					counter.Add(1)
					scope.Go(func() {
						counter.Add(1)
					})
				},
			)
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	assert.Equal(t, int32(4), counter.Load())
}

func TestScopeMultipleGo(t *testing.T) {
	ctx := context.Background()
	counter := atomic.Int32{}
	handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, scope *gather.Scope[int]) (int, error) {
		if in == 2 || in == 3 {
			for range 3 {
				scope.Go(
					func() {
						counter.Add(1)
					},
				)
			}
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	assert.Equal(t, int32(2*3), counter.Load())
}

func TestScopeGoWithRetry(t *testing.T) {
	ctx := context.Background()
	panicCh := make(chan any, 20)

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		if in == 2 || in == 3 {
			scope.Go(
				func() {
					defer func() {
						if r := recover(); r != nil {
							panicCh <- r
						}
					}()
					scope.Retry(ctx, in+1)
				},
			)
			return 0, errors.New("oops")
		}
		return in * 2, nil
	})

	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
	}

	close(panicCh)
	var actualPanics int
	for r := range panicCh {
		assert.Contains(t, r, "Invalid attempt to retry")
		actualPanics++
	}
	assert.Equal(t, 2, actualPanics)
}

func TestScopeGoThenRetry(t *testing.T) {
	ctx := context.Background()
	counter := atomic.Int32{}
	panicCh := make(chan any, 20)

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
			}
		}()
		if in == 2 || in == 3 {
			scope.Go(
				func() {
					counter.Add(1)
				},
			)
			scope.Retry(ctx, in+1)
			return 0, errors.New("oops")
		}
		return in * 2, nil
	})

	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
	}
	close(panicCh)
	var actualPanics int
	for r := range panicCh {
		assert.Contains(t, r, "Invalid attempt to retry")
		actualPanics++
	}
	assert.Equal(t, 2, actualPanics)
	assert.Equal(t, int32(2), counter.Load())
}

func TestScopeRetryThenGo(t *testing.T) {
	ctx := context.Background()
	totalAttempts := atomic.Int32{}
	counter := atomic.Int32{}

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == 2 || in == 3 {
			scope.Retry(ctx, in+1)
			scope.Go(func() {
				counter.Add(1)
			})
			return 0, errors.New("invalid number")
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(5)) {
		got++
	}
	assert.Equal(t, 20, got)
	assert.Equal(t, int32(3), counter.Load())
	assert.Equal(t, int32(23), totalAttempts.Load())
}

func TestOrderedWorkersAndScopeRetryThenGo(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		totalAttempts := atomic.Int32{}
		counter := atomic.Int32{}

		opts := []gather.Opt{
			gather.WithWorkerSize(5),
			gather.WithBufferSize(3),
			gather.WithOrderPreserved(),
		}

		mw := mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second)

		handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
			totalAttempts.Add(1)
			if in == 2 || in == 3 {
				scope.Retry(ctx, in+1)
				scope.Go(func() {
					counter.Add(1)
				})
				return 0, errors.New("invalid number")
			}
			return in * 2, nil
		})

		var got int
		for v := range gather.Workers(ctx, gen(ctx, 1000), mw(handler), opts...) {
			if got != 2 && got != 3 {
				require.Equal(t, got*2, v)
			}
			got++
		}
		assert.Equal(t, 1000, got)
		assert.Equal(t, int32(3), counter.Load())
		assert.Equal(t, int32(1003), totalAttempts.Load())
	})
}

func TestScopeRetryDuringCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	totalAttempts := atomic.Int32{}

	handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == 2 || in == 3 {
			cancel()
			scope.Retry(ctx, in+1)
			return 0, errors.New("invalid number")
		}
		return in * 2, nil
	})

	var got int
	for range gather.Workers(ctx, gen(ctx, 20), handler, gather.WithWorkerSize(2)) {
		got++
	}
	assert.LessOrEqual(t, got, 2)
	assert.LessOrEqual(t, int32(2), totalAttempts.Load())
}
