package shard_test

import (
	"context"
	"gather/internal/shard"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepartitionOneToMany(t *testing.T) {
	in := make(chan int)
	go func() {
		for i := range 100 {
			in <- i
		}
		close(in)
	}()
	ctx := context.Background()
	outs := shard.Repartition[int](10).Apply(ctx, in)
	assert.Len(t, outs, 10)
	seen := make([]bool, 100)
	wg := sync.WaitGroup{}
	for s := range 10 {
		wg.Go(func() {
			for v := range outs[s] {
				assert.False(t, seen[v])
				assert.LessOrEqual(t, 0, v)
				assert.Less(t, v, 100)
				seen[v] = true
			}
		})
	}
	wg.Wait()
}

func TestRepartitionManyToOne(t *testing.T) {
	ins := make([]<-chan int, 10)
	for s := range 10 {
		in := make(chan int)
		ins[s] = in
		go func() {
			for i := range 10 {
				in <- s + 10*i
			}
			close(in)
		}()
	}
	ctx := context.Background()
	outs := shard.Repartition[int](1).Apply(ctx, ins...)
	require.Len(t, outs, 1)
	out := outs[0]

	seen := make([]bool, 100)
	for v := range out {
		assert.False(t, seen[v])
		seen[v] = true
		assert.LessOrEqual(t, 0, v)
		assert.Less(t, v, 100)
	}
}

func TestMultipleRepartitions(t *testing.T) {
	in := make(chan int)
	go func() {
		for i := range 1000 {
			in <- i
		}
		close(in)
	}()
	ctx := context.Background()

	outs1 := shard.Repartition[int](10).Apply(ctx, in)
	assert.Len(t, outs1, 10)
	assert.Equal(t, 1, cap(outs1[0]))
	outs2 := shard.Repartition[int](5).Apply(ctx, outs1...)
	assert.Len(t, outs2, 5)
	assert.Equal(t, 1, cap(outs2[0]))
	outs3 := shard.Repartition[int](20).Apply(ctx, outs2...)
	assert.Len(t, outs3, 20)
	assert.Equal(t, 1, cap(outs3[0]))
	outs4 := shard.Repartition[int](1).Apply(ctx, outs3...)
	require.Len(t, outs4, 1)
	assert.Equal(t, 1, cap(outs4[0]))
	seen := make([]bool, 1000)
	for v := range outs4[0] {
		assert.False(t, seen[v])
		seen[v] = true
		assert.LessOrEqual(t, 0, v)
		assert.Less(t, v, 1000)
	}
}

func TestMultipleRepartitionsWithBuffer(t *testing.T) {
	in := make(chan int)
	go func() {
		for i := range 1000 {
			in <- i
		}
		close(in)
	}()
	ctx := context.Background()

	outs1 := shard.Repartition[int](10).WithBuffer(5).Apply(ctx, in)
	assert.Len(t, outs1, 10)
	assert.Equal(t, 5, cap(outs1[0]))
	outs2 := shard.Repartition[int](1).WithBuffer(20).Apply(ctx, outs1...)
	require.Len(t, outs2, 1)
	assert.Equal(t, 20, cap(outs2[0]))
	seen := make([]bool, 1000)
	for v := range outs2[0] {
		assert.False(t, seen[v])
		seen[v] = true
		assert.LessOrEqual(t, 0, v)
		assert.Less(t, v, 1000)
	}
}

func TestRepartitionWithRoute(t *testing.T) {
	in := make(chan int)
	go func() {
		for i := range 1000 {
			in <- i
		}
		close(in)
	}()
	ctx := context.Background()

	route := func() shard.RouteFunc[int] {
		return func(inShard int, job int, v int) int {
			if v >= 500 {
				return 1
			}
			return 0
		}
	}

	outs := shard.Repartition[int](2).WithRoute(route()).Apply(ctx, in)
	assert.Len(t, outs, 2)
	assert.Equal(t, 1, cap(outs[0]))

	seen := make([]bool, 1000)
	wg := sync.WaitGroup{}
	for shard, out := range outs {
		wg.Go(func() {
			for v := range out {
				assert.False(t, seen[v])
				seen[v] = true
				if shard == 0 {
					assert.LessOrEqual(t, 0, v)
					assert.Less(t, v, 500)
				} else {
					assert.LessOrEqual(t, 500, v)
					assert.Less(t, v, 1000)
				}
			}
		})
	}
	wg.Wait()
}

func TestRepartitionWithNoInputShards(t *testing.T) {
	ctx := context.Background()
	panicCh := make(chan any, 1)
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
				close(panicCh)
			}
		}()
		shard.Repartition[int](10).Apply(ctx)
	})
	wg.Wait()
	assert.Contains(t, <-panicCh, "must send at least 1 input channel")
}
