package shard_test

import (
	"context"
	"fmt"
	"gather"
	"gather/internal/shard"
	"sync"
	"testing"
)

// go test -bench=BenchmarkWorkers ./internal/shard -benchtime=5x
// best results: 10k shards and 10 workers/shard
func BenchmarkWorkers(b *testing.B) {
	workerBuffer := 100
	jobs := 1_000_000
	add := func(num int) gather.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
			return in + num, nil
		}
	}

	shardSizes := []int{1, 100, 1_000, 10_000, 100_000}
	for _, shardSize := range shardSizes {
		// make sure shards*workers is always the same for a fair comparison
		workerSizePerShard := 100_000 / shardSize
		b.Run(fmt.Sprintf("shards: %1.0e, workersPerShard: %1.e", float64(shardSize), float64(workerSizePerShard)), func(b *testing.B) {
			for b.Loop() {
				workerOpts := []gather.Opt{
					gather.WithWorkerSize(workerSizePerShard),
					gather.WithBufferSize(workerBuffer),
				}

				ctx := context.Background()

				// generators (one for each shard)
				ins := make([]<-chan int, shardSize)
				for s := range shardSize {
					in := make(chan int, workerBuffer)
					ins[s] = in
					go func() {
						defer close(in)
						for i := range jobs / shardSize {
							select {
							case <-ctx.Done():
								return
							case in <- s + i*shardSize:
							}
						}
					}()
				}

				// showing multiple stages of sharded workers
				outs1 := shard.Shards(ctx, ins, add(3), workerOpts...)
				outs2 := shard.Shards(ctx, outs1, add(-3), workerOpts...)

				// consume results from all shards
				wg := sync.WaitGroup{}
				for _, shard := range outs2 {
					wg.Go(func() {
						for range shard {
						}
					})
				}
				wg.Wait()
			}
		})
	}
}
