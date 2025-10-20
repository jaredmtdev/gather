/*
This example shows how scope.RetryAfter can be utilized by a middleware to implement a retry logic with max retries.
A real-world implementation would likely have much more options like exponential back-off, jitter, etc.
*/
package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/examples/internal/samplemiddleware"
)

type Job[T any] struct {
	Value   T
	Retries int
}

var ErrOops = errors.New("oops")

func shouldRetryUntilMax[T any](maxRetries int) func(Job[T]) (Job[T], bool) {
	return func(job Job[T]) (Job[T], bool) {
		if job.Retries >= maxRetries {
			return job, false
		}
		job.Retries++
		return job, true
	}
}

func tripple() gather.HandlerFunc[Job[int], int] {
	return func(ctx context.Context, job Job[int], _ *gather.Scope[Job[int]]) (int, error) {
		// simulate work
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-time.After(time.Duration(rand.Intn(100)) * time.Millisecond):
		}

		// random failures
		if rand.Intn(2) == 1 {
			return job.Value * 3, nil
		}
		return 0, ErrOops
	}
}

func main() {
	mw := gather.Chain(
		samplemiddleware.RetryAfter[Job[int], int](200*time.Millisecond, shouldRetryUntilMax[int](3)),
		samplemiddleware.Logger[Job[int], int]("INFO", "ERROR"),
	)

	in := make(chan Job[int])
	go func() {
		for i := range 30 {
			in <- Job[int]{Value: i}
		}
		close(in)
	}()
	for range gather.Workers(context.Background(), in, mw(tripple()), gather.WithWorkerSize(4)) {
	}
}
