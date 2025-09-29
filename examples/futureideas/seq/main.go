/*
the seq package would allow you to work with gather without directly using channels
below is an example of using a slice as input and then iterating through iter.Seq
Note that you can break the loop any time without having to manually cancel the context or close any channel
*/
package main

import (
	"context"
	"fmt"
	"slices"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/internal/seq"
)

func main() {
	ctx := context.Background()
	vals := make([]int, 100)
	for i := range vals {
		vals[i] = i
	}
	add := func(num int) gather.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
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

	// collect remaining values
	fmt.Println(slices.Collect(out))
}
