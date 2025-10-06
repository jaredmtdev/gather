package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/examples/internal/samplegen"
	"github.com/jaredmtdev/gather/examples/internal/samplemiddleware"
)

func main() {
	mw := gather.Chain(
		samplemiddleware.Timeout[int, int](170*time.Millisecond),
		samplemiddleware.Logger[int, int]("INFO", "ERROR"),
		samplemiddleware.Counter[int, int](15),
		samplemiddleware.RetryAfter[int, int](100*time.Millisecond),
	)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	wg := sync.WaitGroup{}
	addHandler := func(num int) gather.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, _ *gather.Scope[int]) (int, error) {
			select {
			case <-time.After(time.Duration(rand.Intn(200)) * time.Millisecond):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			if in == 39 {
				// much more likely for time out error at the end
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(125 * time.Millisecond):
				}
			}

			// uncomment to test early cancel
			// if in == 10 {
			// 	cancel()
			// }

			if in == 7 {
				wg.Go(func() {
					time.Sleep(2 * time.Second)
					fmt.Println("safely executed from new go routine!")
				})
			}
			return num + in, nil
		}
	}

	opts := []gather.Opt{
		gather.WithWorkerSize(2),
		gather.WithBufferSize(0),
	}

	for range gather.Workers(ctx, samplegen.Range(40), mw(addHandler(3)), opts...) {
	}
	wg.Wait()
	fmt.Println("done!")
	time.Sleep(time.Second)
	fmt.Println("shutting down")
}
