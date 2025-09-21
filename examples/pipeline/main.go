package main

import (
	"context"
	"fmt"
	"gather"
	"gather/examples/internal/samplegen"
	"gather/examples/internal/samplemiddleware"
	"math/rand"
	"time"
)

func main() {
	mw := samplemiddleware.Logger[int, int]("INFO", "ERROR")
	addHandler := func(num int) gather.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, scope *gather.Scope[int]) (int, error) {
			select {
			case <-time.After(time.Duration(rand.Intn(200)) * time.Millisecond):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			return num + in, nil
		}
	}
	wrappedAddHandler := func(num int) gather.HandlerFunc[int, int] {
		return mw(addHandler(num))
	}
	ctx := context.Background()

	// example of a pipeline
	out1 := gather.Workers(ctx, samplegen.Range(40), wrappedAddHandler(1), gather.WithWorkerSize(5))
	out2 := gather.Workers(ctx, out1, wrappedAddHandler(2), gather.WithWorkerSize(3))
	for range gather.Workers(ctx, out2, wrappedAddHandler(3), gather.WithWorkerSize(2), gather.WithBufferSize(2)) {
	}

	// should not see anymore noise from the pipeline at this point
	fmt.Println("done!")
	time.Sleep(time.Second)
	fmt.Println("shutting down")
}
