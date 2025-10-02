package shard

import (
	"context"
	"sync"

	"github.com/jaredmtdev/gather/internal/op"
)

// RouteFunc - used to determine which shard to send data to.
type RouteFunc[T any] func(inShard int, job int, v T) (outShard int)

// Repartitioner - used to configure a repartition
//
// repartitioning cannot guarantee order.
type Repartitioner[T any] struct {
	partitionSize int
	bufferSize    *int
	route         RouteFunc[T]
	outsSendTo    []chan T
	outsResponse  []<-chan T
}

// WithBuffer - option to set new buffer size. by default will choose same buffer as input data.
func (r *Repartitioner[T]) WithBuffer(bufferSize int) *Repartitioner[T] {
	r.bufferSize = &bufferSize
	return r
}

// WithRoute - option to implement which outgoing shard to route to
// by default: uses round-robin.
func (r *Repartitioner[T]) WithRoute(route RouteFunc[T]) *Repartitioner[T] {
	r.route = route
	return r
}

func (r *Repartitioner[T]) repartitionShard(ctx context.Context, inShard int, in <-chan T) {
	var outShard int
	var value T
	for job := 0; ; job++ {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-in:
			if !ok {
				return
			}
			outShard = op.PosMod(r.route(inShard, job, v), r.partitionSize)
			value = v
		}

		select {
		case <-ctx.Done():
			return
		case r.outsSendTo[outShard] <- value:
		}
	}
}

// Apply - runs the repartition.
func (r *Repartitioner[T]) Apply(ctx context.Context, ins ...<-chan T) []<-chan T {
	if len(ins) == 0 {
		panic("must send at least 1 input channel to Repartition")
	}
	if r.bufferSize == nil {
		bufferSize := max(cap(ins[0]), 1)
		r.bufferSize = &bufferSize
	}

	r.outsResponse = make([]<-chan T, r.partitionSize)
	r.outsSendTo = make([]chan T, r.partitionSize)
	for i := range r.outsResponse {
		r.outsSendTo[i] = make(chan T, *r.bufferSize)
		r.outsResponse[i] = r.outsSendTo[i]
	}

	wgInShard := sync.WaitGroup{}
	for inShard, in := range ins {
		wgInShard.Go(func() {
			r.repartitionShard(ctx, inShard, in)
		})
	}

	go func() {
		wgInShard.Wait()
		for i := range r.outsSendTo {
			close(r.outsSendTo[i])
		}
	}()

	return r.outsResponse
}

// Repartition - configure a repartition
// used to change N input channels into M output channels.
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
