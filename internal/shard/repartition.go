package shard

import (
	"context"
	"sync"
)

// Hash - used to determine which shard to send data to
type Hash[T any] func(inShard int, job int, v T) (outShard int)

// RepartitionBuilder - used to configure a repartition
type RepartitionBuilder[T any] struct {
	partitions int
	bufferSize *int
	h          Hash[T]
}

// WithBufferSize - option to set new buffer size. by default will choose same buffer as input data
func (pb *RepartitionBuilder[T]) WithBufferSize(bufferSize int) *RepartitionBuilder[T] {
	pb.bufferSize = &bufferSize
	return pb
}

// WithHash - option to use a custom hash function to design which partition to send data to
func (pb *RepartitionBuilder[T]) WithHash(h Hash[T]) *RepartitionBuilder[T] {
	pb.h = h
	return pb
}

// Run - runs the repartition to get the new partitions
// note that repartitioning cannot guarantee order
func (pb *RepartitionBuilder[T]) Run(ctx context.Context, ins ...<-chan T) []<-chan T {
	if len(ins) == 0 {
		panic("must send at least 1 input channel to Repartition")
	}
	if pb.bufferSize == nil {
		bufferSize := cap(ins[0])
		pb.bufferSize = &bufferSize
	}

	outs := make([]<-chan T, pb.partitions)
	repartitioned := make([]chan T, pb.partitions)
	for i := range outs {
		repartitioned[i] = make(chan T, *pb.bufferSize)
		outs[i] = repartitioned[i]
	}

	wgInShard := sync.WaitGroup{}
	for inShard, in := range ins {
		wgInShard.Go(func() {
			job := 0
			for v := range in {
				outShard := pb.h(inShard, job, v) % pb.partitions
				job++
				select {
				case <-ctx.Done():
					return
				case repartitioned[outShard] <- v:
				}
			}
		})
	}

	go func() {
		wgInShard.Wait()
		for i := range outs {
			close(repartitioned[i])
		}
	}()

	return outs
}

// Repartition - configure a repartition
// used to change N input channels into M output channels
func Repartition[T any](partitions int) *RepartitionBuilder[T] {
	defaultHash := func(inShard int, job int, v T) int {
		return inShard + job
	}
	p := &RepartitionBuilder[T]{
		partitions: partitions,
		h:          defaultHash,
	}
	return p
}
