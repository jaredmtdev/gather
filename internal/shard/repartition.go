package shard

import (
	"context"
	"gather/internal/op"
	"sync"
)

// RouteFunc - used to determine which shard to send data to
type RouteFunc[T any] func(inShard int, job int, v T) (outShard int)

// Repartitioner - used to configure a repartition
//
// repartitioning cannot guarantee order
type Repartitioner[T any] struct {
	partitionSize int
	bufferSize    *int
	route         RouteFunc[T]
}

// WithBuffer - option to set new buffer size. by default will choose same buffer as input data
func (r *Repartitioner[T]) WithBuffer(bufferSize int) *Repartitioner[T] {
	r.bufferSize = &bufferSize
	return r
}

// WithRoute - option to implement which outgoing shard to route to
// by default: uses round-robin
func (r *Repartitioner[T]) WithRoute(route RouteFunc[T]) *Repartitioner[T] {
	r.route = route
	return r
}

// Apply - runs the repartition
func (r *Repartitioner[T]) Apply(ctx context.Context, ins ...<-chan T) []<-chan T {
	if len(ins) == 0 {
		panic("must send at least 1 input channel to Repartition")
	}
	if r.bufferSize == nil {
		bufferSize := max(cap(ins[0]), 1)
		r.bufferSize = &bufferSize
	}

	outsResponse := make([]<-chan T, r.partitionSize)
	outsSendTo := make([]chan T, r.partitionSize)
	for i := range outsResponse {
		outsSendTo[i] = make(chan T, *r.bufferSize)
		outsResponse[i] = outsSendTo[i]
	}

	wgInShard := sync.WaitGroup{}
	for inShard, in := range ins {
		wgInShard.Go(func() {
			job := 0
			for v := range in {
				outShard := op.PosMod(r.route(inShard, job, v), r.partitionSize)
				job++
				select {
				case <-ctx.Done():
					return
				case outsSendTo[outShard] <- v:
				}
			}
		})
	}

	go func() {
		wgInShard.Wait()
		for i := range outsSendTo {
			close(outsSendTo[i])
		}
	}()

	return outsResponse
}

// Repartition - configure a repartition
// used to change N input channels into M output channels
func Repartition[T any](newPartitionSize int) *Repartitioner[T] {
	if newPartitionSize <= 0 {
		panic("must have at least 1 partition")
	}
	r := &Repartitioner[T]{
		partitionSize: newPartitionSize,
		route: func(inShard int, job int, _ T) int {
			// default: round-robin
			return inShard + job
		},
	}
	return r
}
