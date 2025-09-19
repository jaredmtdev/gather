/*
this example shows how the experimental shard package can be used
*/
package main

import (
	"context"
	"fmt"
	"together"
	"together/internal/shard"
)

func main() {
	jobs := 10_000
	shardSize := 1_000

	workerSizePerShard := 100
	workerBuffer := 100

	workerOpts := []together.Opt{
		together.WithWorkerSize(workerSizePerShard),
		together.WithBufferSize(workerBuffer),
		//together.WithOrderPreserved(),
	}

	ctx := context.Background()
	add := func(num int) together.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *together.Scope[int]) (int, error) {
			return in + num, nil
		}
	}

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
	outs1 := shard.Workers(ctx, ins, add(3), workerOpts...)
	outs2 := shard.Workers(ctx, outs1, add(-3), workerOpts...)

	for v := range shard.Repartition[int](1).Run(ctx, outs2...)[0] {
		fmt.Printf("%v ", v)
	}
	fmt.Println("")

	//// any order
	//wgShard := sync.WaitGroup{}
	//for i, shard := range outs2 {
	//	wgShard.Go(func() {
	//		for v := range shard {
	//			fmt.Printf("shard %v: %v\n", i, v)
	//		}
	//	})
	//}
	//wgShard.Wait()

	//	// preserve order via round robin
	//	var done bool
	//
	// loop:
	//
	//	for !done {
	//		for i := range len(outs2) {
	//			select {
	//			case <-ctx.Done():
	//				break loop
	//			case v, ok := <-outs2[i]:
	//				if !ok {
	//					done = true
	//					break
	//				}
	//				fmt.Printf("shard %v: %v\n", i, v)
	//			}
	//		}
	//	}
}
