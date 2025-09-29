/*
This example shows how you can build more conveniences when input/output types are the same for all stages
A reusable "Pipe" helper can be designed so that composing pipelines becomes very easy to do and readable
*/

package main

import (
	"context"
	"fmt"
	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/examples/internal/samplegen"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

type WorkerBuilder[IN any, OUT any] func(ctx context.Context, in <-chan IN, f func(IN) OUT) <-chan OUT

// Pipe - lets you chain stages together
func Pipe[T any](in <-chan T, stages ...func(in <-chan T) <-chan T) <-chan T {
	out := in
	for _, stage := range stages {
		out = stage(out)
	}
	return out
}

// op - helper for doing math operations
// this is just for making the usage more readable
type op[T any] struct {
	stageFunc func(in <-chan T) <-chan T
}

func (o *op[T]) To(in <-chan T) <-chan T {
	return o.stageFunc(in)
}
func (o *op[T]) From(in <-chan T) <-chan T {
	return o.stageFunc(in)
}
func (o *op[T]) By(in <-chan T) <-chan T {
	return o.stageFunc(in)
}

func main() {
	ctx := context.Background()

	opts := []gather.Opt{
		gather.WithWorkerSize(runtime.GOMAXPROCS(0)),
		gather.WithBufferSize(runtime.GOMAXPROCS(0)),
		gather.WithOrderPreserved(),
	}

	buildWorkers := WorkerBuilder[int, int](func(ctx context.Context, in <-chan int, f func(v int) int) <-chan int {
		pool := sync.Pool{New: func() any { return rand.New(rand.NewSource(time.Now().UnixNano())) }}
		handler := gather.HandlerFunc[int, int](func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
			r := pool.Get().(*rand.Rand)
			defer pool.Put(r)
			time.Sleep(time.Duration(r.Intn(200)) * time.Millisecond)
			return f(in), nil
		})
		return gather.Workers(ctx, in, handler, opts...)
	})

	add := func(num int) *op[int] {
		return &op[int]{
			stageFunc: func(in <-chan int) <-chan int {
				return buildWorkers(ctx, in, func(v int) int {
					return v + num
				})
			},
		}
	}
	subtract := func(num int) *op[int] {
		return &op[int]{
			stageFunc: func(in <-chan int) <-chan int {
				return buildWorkers(ctx, in, func(v int) int {
					return v - num
				})
			},
		}
	}
	multiply := func(num int) *op[int] {
		return &op[int]{
			stageFunc: func(in <-chan int) <-chan int {
				return buildWorkers(ctx, in, func(v int) int {
					return v * num
				})
			},
		}
	}

	// === example usage ===

	fmt.Println("pipe stages chained manually")
	for v := range subtract(3).From(multiply(2).By(add(5).To(samplegen.Range(30)))) {
		fmt.Printf("%v ", v)
	}
	fmt.Println("")

	fmt.Println("pipe stages chained with Pipe helper")
	out := Pipe(
		samplegen.Range(30),
		add(5).To,
		multiply(2).By,
		subtract(3).From,
	)
	for v := range out {
		fmt.Printf("%v ", v)
	}
	fmt.Println("")
}
