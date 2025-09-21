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
	outs2 := shard.Repartition[int](5).Apply(ctx, outs1...)
	assert.Len(t, outs2, 5)
	outs3 := shard.Repartition[int](20).Apply(ctx, outs2...)
	assert.Len(t, outs3, 20)
	outs4 := shard.Repartition[int](1).Apply(ctx, outs3...)
	require.Len(t, outs4, 1)
	seen := make([]bool, 1000)
	for v := range outs4[0] {
		assert.False(t, seen[v])
		seen[v] = true
		assert.LessOrEqual(t, 0, v)
		assert.Less(t, v, 1000)
	}
}
