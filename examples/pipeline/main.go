package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"together/examples/internal/samplegen"
	"together/examples/internal/samplemiddleware"
	"together/pkg/together"
)

func main() {
	mw := samplemiddleware.Logger[int, int]("INFO", "ERROR")
	addHandler := func(num int) together.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, scope *together.Scope[int]) (int, error) {
			select {
			case <-time.After(time.Duration(rand.Intn(200)) * time.Millisecond):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			return num + in, nil
		}
	}
	wrappedAddHandler := func(num int) together.HandlerFunc[int, int] {
		return mw(addHandler(num))
	}
	ctx := context.Background()

	// example of a pipeline
	out1 := together.Workers(ctx, samplegen.Range(40), wrappedAddHandler(1), together.WithWorkerSize(5))
	out2 := together.Workers(ctx, out1, wrappedAddHandler(2), together.WithWorkerSize(3))
	for range together.Workers(ctx, out2, wrappedAddHandler(3), together.WithWorkerSize(2), together.WithBufferSize(2)) {
	}

	// should not see anymore noise from the pipeline at this point
	fmt.Println("done!")
	time.Sleep(time.Second)
	fmt.Println("shutting down")
}
