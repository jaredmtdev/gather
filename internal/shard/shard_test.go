package shard_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/internal/shard"
	"github.com/stretchr/testify/assert"
)

func TestApply(t *testing.T) {
	ctx := context.Background()
	ins := make([]<-chan int, 5)
	for i := range ins {
		inCh := make(chan int, 1)
		ins[i] = inCh
		inCh <- 1
		close(inCh)
	}
	handler := func() gather.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
			return in, nil
		}
	}
	outs := shard.Apply(ctx, ins, handler())
	var total int
	for _, out := range outs {
		for v := range out {
			assert.Equal(t, 1, v)
			total += v
		}
	}
	assert.Equal(t, 5, total)
}

func TestApplyWithNoBuffer(t *testing.T) {
	ctx := context.Background()
	ins := make([]<-chan int, 5)
	wgGen := sync.WaitGroup{}
	for i := range ins {
		wgGen.Add(1)
		go func() {
			inCh := make(chan int)
			ins[i] = inCh
			wgGen.Done() // just need to make sure all channels are set before sending to shards
			inCh <- 1
			close(inCh)
		}()
	}
	handler := func() gather.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
			return in, nil
		}
	}
	wgGen.Wait()
	outs := shard.Apply(ctx, ins, handler(), gather.WithBufferSize(0))
	assert.Len(t, outs, 5)

	var total atomic.Int32
	wg := sync.WaitGroup{}
	for _, out := range outs {
		wg.Go(func() {
			for v := range out {
				assert.Equal(t, 1, v)
				total.Add(1)
			}
		})
	}
	wg.Wait()
	assert.Equal(t, int32(5), total.Load())
}

func TestApplyWithMultipleWorkersAndNoBuffer(t *testing.T) {
	ctx := context.Background()
	ins := make([]<-chan int, 5)
	wgGen := sync.WaitGroup{}
	for i := range ins {
		wgGen.Add(1)
		go func() {
			inCh := make(chan int)
			ins[i] = inCh
			wgGen.Done() // just need to make sure all channels are set before sending to shards
			inCh <- 1
			close(inCh)
		}()
	}
	handler := func() gather.HandlerFunc[int, int] {
		return func(_ context.Context, in int, _ *gather.Scope[int]) (int, error) {
			return in, nil
		}
	}
	wgGen.Wait()
	outs := shard.Apply(ctx, ins, handler(), gather.WithBufferSize(0), gather.WithWorkerSize(2))
	assert.Len(t, outs, 5)
	var total atomic.Int32
	wg := sync.WaitGroup{}
	for _, out := range outs {
		wg.Go(func() {
			for v := range out {
				assert.Equal(t, 1, v)
				total.Add(1)
			}
		})
	}
	wg.Wait()
	assert.Equal(t, int32(5), total.Load())
}
