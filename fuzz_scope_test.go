package gather_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/internal/op"
	"github.com/stretchr/testify/assert"
)

func RunFuzzScopeRetryAfterWhenNoError(t *testing.T, workerSize, bufferSize, jobs, retryOn int) {
	t.Helper()
	ctx := context.Background()
	totalAttempts := atomic.Int64{}
	delayUntilRetry := 1 * time.Millisecond
	panicCh := make(chan any, 1)

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == retryOn {
			scope.RetryAfter(ctx, in+1, delayUntilRetry)
			return 0, nil
		}
		return in * 2, nil
	})

	start := time.Now()
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			panicCh <- recover()
		}()
		var got int
		for range gather.Workers(ctx, gen(ctx, jobs, bufferSize), handler, gather.WithWorkerSize(workerSize), gather.WithBufferSize(bufferSize)) {
			got++
		}
		duration := time.Since(start)
		assert.Equal(t, jobs, got)
		var additionalAttempts int
		if jobs > 0 && retryOn >= 0 {
			additionalAttempts++
		}
		if jobs > 0 {
			assert.LessOrEqual(t, delayUntilRetry.Milliseconds(), duration.Milliseconds())
		}
		assert.Equal(t, int64(jobs+additionalAttempts), totalAttempts.Load())
	})
	wg.Wait()
	close(panicCh)
	r, ok := <-panicCh
	assert.True(t, ok)
	if workerSize < 1 || bufferSize < 0 {
		assert.NotNil(t, r)
	} else {
		assert.Nil(t, r)
	}
}

func FuzzScopeRetryAfterWhenNoError(f *testing.F) {
	f.Add(1, 0, 1, 0)
	f.Add(10, 10, 100, 0)
	f.Add(1000, 1, 1000, 999)

	f.Fuzz(func(t *testing.T, workerSize, bufferSize, jobs, retryOn int) {
		jobs = op.PosMod(jobs, 1_000_000)
		retryOn = op.PosMod(retryOn, max(jobs, 1))
		t.Run(fmt.Sprintf("workerSize %v bufferSize %v jobs %v retryOn %v", workerSize, bufferSize, jobs, retryOn), func(t *testing.T) {
			RunFuzzScopeRetryAfterWhenNoError(t, workerSize, bufferSize, jobs, retryOn)
		})
	})
}
