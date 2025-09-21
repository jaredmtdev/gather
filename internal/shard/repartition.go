package shard

import (
	"context"
	"sync"
)

// PartitionFunc - used to determine which shard to send data to
type PartitionFunc[T any] func(inShard int, job int, v T) (outShard int)

// Repartitioner - used to configure a repartition
//
// note:
// - repartitioning cannot guarantee order
type Repartitioner[T any] struct {
	partitionSize int
	bufferSize    *int
	partition     PartitionFunc[T]
}

// WithBuffer - option to set new buffer size. by default will choose same buffer as input data
func (r *Repartitioner[T]) WithBuffer(bufferSize int) *Repartitioner[T] {
	r.bufferSize = &bufferSize
	return r
}

// WithHash - option to use a custom hash function to design which shard to send data to
func (r *Repartitioner[T]) WithHash(pf PartitionFunc[T]) *Repartitioner[T] {
	r.partition = pf
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
				outShard := r.partition(inShard, job, v) % r.partitionSize
				if outShard < 0 {
					outShard *= -1
				}
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
		partition: func(inShard int, job int, _ T) int {
			// default: round-robin
			return inShard + job
		},
	}
	return r
}
