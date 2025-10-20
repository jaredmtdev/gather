/*
This example is a pipeline where all 3 stages are sending errors to a separate error channel.
This pattern might be useful to unblock workers while a separate go routine (or worker pool) prepares a list of errors for an api response.
*/
package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/examples/errchan/errs"
	"github.com/jaredmtdev/gather/examples/internal/samplegen"
)

func main() {
	errCh := make(chan error)
	wg := sync.WaitGroup{}
	addHandler := func(num int) gather.HandlerFunc[int, int] {
		return func(ctx context.Context, in int, _ *gather.Scope[int]) (int, error) {
			select {
			case <-time.After(time.Duration(rand.Intn(200)) * time.Millisecond):
			case <-ctx.Done():
				return 0, ctx.Err()
			}
			if in == 15 || in == 16 {
				err := errs.NewErrInvalidNumber(in)
				wg.Go(func() {
					select {
					case <-ctx.Done():
					case errCh <- err:
					}
				})
				return 0, err
			}
			return num + in, nil
		}
	}
	ctx := context.Background()

	// all 3 stages are sending errors here to be handled
	var errs []error
	wgErrs := sync.WaitGroup{}
	wgErrs.Go(func() {
		for err := range errCh {
			errs = append(errs, err)
		}
	})

	// three stage pipeline
	out1 := gather.Workers(ctx, samplegen.Range(40), addHandler(1), gather.WithWorkerSize(5))
	out2 := gather.Workers(ctx, out1, addHandler(2), gather.WithWorkerSize(3))
	for v := range gather.Workers(ctx, out2, addHandler(3), gather.WithWorkerSize(2), gather.WithBufferSize(2)) {
		fmt.Printf("%v ", v)
	}
	wg.Wait()
	close(errCh)
	wgErrs.Wait()
	fmt.Printf("\nerrors: %v\n", errs)

	// should not see anymore noise from the pipeline at this point
	fmt.Println("done!")
	time.Sleep(time.Second)
	fmt.Println("shutting down")
}
