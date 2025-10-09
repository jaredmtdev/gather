//go:build clusterfuzzlite
// +build clusterfuzzlite

package fuzzgather

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/internal/op"
)

func FuzzScopeRetryAfterWhenNoError(data []byte) int {
	// Expect 4 int32s: workerSize, bufferSize, jobs, retryOn
	if len(data) < 16 {
		return 0
	}

	workerSize := decode(data[0:4])
	bufferSize := decode(data[4:8])
	jobs := decode(data[8:12])
	retryOn := decode(data[12:16])

	jobs = op.PosMod(jobs, 1_000_000)
	retryOn = op.PosMod(retryOn, max(jobs, 1))

	if bufferSize > 1_000_000 || workerSize > 1_000_000 || bufferSize < -50 || workerSize < -50 {
		return 0
	}

	fmt.Printf("workerSize %v bufferSize %v jobs %v retryOn %v\n", workerSize, bufferSize, jobs, retryOn)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	panicExpected := workerSize < 1 || bufferSize < 0
	var panicFound bool

	totalAttempts := atomic.Int32{}
	delayUntilRetry := 1 * time.Millisecond

	handler := gather.HandlerFunc[int, int](func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
		totalAttempts.Add(1)
		if in == retryOn {
			scope.RetryAfter(ctx, in+1, delayUntilRetry)
			return 0, nil
		}
		return in * 2, nil
	})

	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				panicFound = true
				if panicExpected {
					return
				}
				panic(r)
			}
		}()

		out := gather.Workers(ctx,
			gen(ctx, jobs, max(bufferSize, 0)),
			handler,
			gather.WithWorkerSize(workerSize),
			gather.WithBufferSize(bufferSize),
		)

		for {
			select {
			case <-ctx.Done():
				return
			case _, ok := <-out:
				if !ok {
					return
				}
			}
		}
	})
	wg.Wait()

	if panicExpected && !panicFound {
		panic("expected a panic. No panic occurred")
	}

	if ctx.Err() != nil {
		return 0
	}

	return 1
}

func gen(ctx context.Context, jobs, buf int) <-chan int {
	ch := make(chan int, buf)
	go func() {
		defer close(ch)
		for i := range jobs {
			select {
			case <-ctx.Done():
				return
			case ch <- i:
			}
		}
	}()
	return ch
}

func decode(data []byte) int {
	v, _ := binary.Varint(data)
	if v > math.MaxInt {
		v = v % math.MaxInt
	} else if v < math.MinInt {
		v = v % math.MinInt
	}
	return int(v)
}
