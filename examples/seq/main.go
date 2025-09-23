package main

import (
	"context"
	"fmt"
	"gather"
	"gather/internal/seq"
	"slices"
)

func main() {
	ctx := context.Background()
	vals := make([]int, 100)
	for i := range vals {
		vals[i] = i
	}
	add := func(num int) gather.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, _ *gather.Scope[int]) (int, error) {
			return in + num, nil
		}
	}
	buffer := 5
	out := seq.Workers(
		ctx,
		slices.Values(vals),
		add(3),
		buffer,
		gather.WithWorkerSize(5),
		gather.WithOrderPreserved(),
	)
	for v := range out {
		if v >= 50 {
			break
		}
		fmt.Printf("%v ", v)
	}
	fmt.Println("")
}
