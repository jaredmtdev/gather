package main

import (
	"context"
	"fmt"
	"together"
	"together/examples/internal/samplegen"
)

// wraps around Workers by sharding input out to multiple shards
// each shard gets it's own worker pool
func WorkersSharded[IN, OUT any](
	ctx context.Context,
	in <-chan IN,
	mapToShard func(IN) uint,
	shardSize int,
	bufferSize int,
	handler together.HandlerFunc[IN, OUT],
	workerOpts ...together.Opt,
) []<-chan OUT {
	if shardSize < 1 {
		shardSize = 1
	}
	ins := make([]chan IN, shardSize)
	outs := make([]<-chan OUT, shardSize)

	// start each shard
	for i := range ins {
		ins[i] = make(chan IN, bufferSize)
		outs[i] = together.Workers(ctx, ins[i], handler, workerOpts...)
	}

	// send to each shard
	go func() {
		defer func() {
			for i := range ins {
				close(ins[i])
			}
		}()
		for v := range in {
			shard := int(mapToShard(v) % uint(shardSize))
			select {
			case <-ctx.Done():
				return
			case ins[shard] <- v:
			}
		}
	}()

	return outs
}

func main() {
	jobs := 10_000
	shardSize := 1000
	shardBuffer := 100

	workerSizePerShard := 10
	workerBuffer := 100

	workerOpts := []together.Opt{
		together.WithWorkerSize(workerSizePerShard),
		together.WithBufferSize(workerBuffer),
		together.WithOrderPreserved(),
	}

	ctx := context.Background()
	shardHash := func(v int) uint {
		return uint(v)
	}
	add := func(num int) together.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *together.Scope[int]) (int, error) {
			return in + num, nil
		}
	}
	shards := WorkersSharded(ctx, samplegen.Range(jobs, shardBuffer), shardHash, shardSize, shardBuffer, add(3), workerOpts...)

	// any order
	//wgShard := sync.WaitGroup{}
	//for i, shard := range shards {
	//	wgShard.Go(func() {
	//		for v := range shard {
	//			fmt.Printf("shard %v: %v\n", i, v)
	//		}
	//	})
	//}
	//wgShard.Wait()

	// preserve order via round robin
	var done bool
	for !done {
		for i := range len(shards) {
			select {
			case v, ok := <-shards[i]:
				if !ok {
					done = true
					continue
				}
				fmt.Printf("shard %v: %v\n", i, v)
			}
		}
	}
}
