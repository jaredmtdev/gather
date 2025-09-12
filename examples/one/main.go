package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"
	"together/examples/internal/samplemiddleware"
	"together/pkg/together"
)

func main() {
	gen := func(n int) chan int {
		out := make(chan int)
		go func() {
			for i := range n {
				out <- i
			}
			close(out)
		}()
		return out
	}

	mw := together.Chain(
		samplemiddleware.Timeout[int, int](170*time.Millisecond),
		samplemiddleware.Logger[int, int]("INFO", "ERROR"),
		samplemiddleware.Counter[int, int](15),
		samplemiddleware.RetryAfter[int, int](100*time.Millisecond),
	)

	add := func(num int) together.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, scope *together.Scope[int]) (int, error) {
			time.Sleep(time.Duration(rand.Int63n(200)) * time.Millisecond)
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			default:
			}
			// if in == 7 {
			// 	handler.RetryAfter(in+1, 800*time.Millisecond)
			// 	err := errors.New("oh no!")
			// 	//fmt.Println("error", err)
			// 	return 0, err
			// }
			if in == 39 {
				// much more likely for time out error at the end
				select {
				case <-ctx.Done():
					return 0, ctx.Err()
				case <-time.After(100 * time.Millisecond):
				}
			}
			if in == 7 {
				scope.Go(func() {
					time.Sleep(2 * time.Second)
					fmt.Println("safely executed from new go routine!")
				})
			}
			return num + in, nil
		}
	}

	opts := []together.Opt{
		together.WithWorkerSize(2),
		together.WithBufferSize(0),
	}

	for range together.Workers(context.Background(), gen(40), mw(add(3)), opts...) {
	}

}
