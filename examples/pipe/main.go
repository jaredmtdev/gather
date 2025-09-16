/*
This example shows how you can build more conveniences when input/output types are the same for all stages

The Pipe helper is built to make it look very easy to understand
*/

package main

import (
	"context"
	"fmt"
	"together"
	"together/examples/internal/samplegen"
)

type WorkerBuilder[IN any, OUT any] func(ctx context.Context, in <-chan IN, f func(IN) OUT) <-chan OUT

func Pipe[T any](in <-chan T, stages ...func(in <-chan T) <-chan T) <-chan T {
	if len(stages) == 0 {
		panic("must have at least 1 stage!")
	}
	out := in
	for _, stage := range stages {
		out = stage(out)
	}
	return out
}

func main() {
	ctx := context.Background()

	buildWorkers := WorkerBuilder[int, int](func(ctx context.Context, in <-chan int, f func(v int) int) <-chan int {
		handler := together.HandlerFunc[int, int](func(_ context.Context, in int, _ *together.Scope[int]) (int, error) {
			return f(in), nil
		})
		return together.Workers(ctx, in, handler, together.WithWorkerSize(4), together.WithBufferSize(2))
	})

	add := func(num int) func(in <-chan int) <-chan int {
		return func(in <-chan int) <-chan int {
			return buildWorkers(ctx, in, func(v int) int {
				return v + num
			})
		}
	}
	subtract := func(num int) func(in <-chan int) <-chan int {
		return func(in <-chan int) <-chan int {
			return buildWorkers(ctx, in, func(v int) int {
				return v - num
			})
		}
	}
	multiply := func(num int) func(in <-chan int) <-chan int {
		return func(in <-chan int) <-chan int {
			return buildWorkers(ctx, in, func(v int) int {
				return v * num
			})
		}
	}

	// TODO: give option to preserve order

	fmt.Println("pipe stages together manually")
	for v := range subtract(3)(multiply(2)(add(5)(samplegen.Range(20)))) {
		fmt.Println(v)
	}

	fmt.Println("pipe stages together with Pipe helper")
	out := Pipe(
		samplegen.Range(20),
		add(5),
		multiply(2),
		subtract(3),
	)
	for v := range out {
		fmt.Println(v)
	}

}
