package together_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"together"
)

/*
expecting the overhead to be significant in this benchmark since the jobs each have negligible cost
go test -bench=BenchmarkWorkersInstantJobs -benchtime=5x

best result: 3 workers and 1500 buffer size.
*/
func BenchmarkWorkersInstantJobs(b *testing.B) {
	maxprocs := runtime.GOMAXPROCS(0)
	bufferSizes := []int{100, 500, 1_000, 1_500, 2_000}
	for workerSize := 1; workerSize <= maxprocs-2; workerSize++ {
		for _, bufferSize := range bufferSizes {
			b.Run(fmt.Sprintf("workers: %v, buffer: %1.1e ", float64(workerSize), float64(bufferSize)), func(b *testing.B) {
				for b.Loop() {
					ctx := context.Background()
					opts := []together.Opt{
						together.WithWorkerSize(workerSize),
						together.WithBufferSize(bufferSize),
					}
					for range together.Workers(ctx, gen(ctx, 10_000_000, bufferSize), add(5), opts...) {
					}
				}
			})
		}
	}
}

/*
go test -bench=BenchmarkWorkersWithSimulatedWork -benchtime=10x

best result: 10k workers and 100k buffer.
*/
func BenchmarkWorkersWithSimulatedWork(b *testing.B) {
	mw := mwRandomDelay[int, int](time.Now().UnixNano(), 0, 20*time.Millisecond)
	bufferSizes := []int{1_000, 10_000, 100_000}
	workerSizes := []int{1_000, 10_000, 100_000}
	for _, workerSize := range workerSizes {
		for _, bufferSize := range bufferSizes {
			b.Run(fmt.Sprintf("workers: %1.0e, buffer: %1.1e ", float64(workerSize), float64(bufferSize)), func(b *testing.B) {
				for b.Loop() {
					ctx := context.Background()
					opts := []together.Opt{
						together.WithWorkerSize(workerSize),
						together.WithBufferSize(bufferSize),
					}
					for range together.Workers(ctx, gen(ctx, 10_000, bufferSize), mw(add(5)), opts...) {
					}
				}
			})
		}
	}
}
