package main

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/examples/internal/samplemiddleware"
)

type Job struct {
	Value   int
	Retries int
}

var ErrOops = errors.New("oops")

func shouldRetryUntilMax(maxRetries int) func(Job) (Job, bool) {
	return func(job Job) (Job, bool) {
		if job.Retries >= maxRetries {
			return job, false
		}
		job.Retries++
		return job, true
	}
}

func tripple() gather.HandlerFunc[Job, int] {
	return func(ctx context.Context, job Job, _ *gather.Scope[Job]) (int, error) {
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
		samplemiddleware.RetryAfter[Job, int](200*time.Millisecond, shouldRetryUntilMax(3)),
		samplemiddleware.Logger[Job, int]("INFO", "ERROR"),
	)

	in := make(chan Job)
	go func() {
		for i := range 30 {
			in <- Job{Value: i}
		}
		close(in)
	}()
	for range gather.Workers(context.Background(), in, mw(tripple()), gather.WithWorkerSize(4)) {
	}
}
