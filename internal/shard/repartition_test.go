package shard_test

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jaredmtdev/gather/internal/shard"
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

func TestRepartitionWithNoPartitions(t *testing.T) {
	ctx := context.Background()
	panicCh := make(chan any, 1)
	in := make(chan int)
	defer close(in)
	wg := sync.WaitGroup{}
	wg.Go(func() {
		defer func() {
			if r := recover(); r != nil {
				panicCh <- r
				close(panicCh)
			}
		}()
		shard.Repartition[int](0).Apply(ctx, in)
	})
	wg.Wait()
	assert.Contains(t, <-panicCh, "must have at least 1 partition")
}

func TestRepartitionOneToManyEarlyCancel(t *testing.T) {
	in := make(chan int, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		defer close(in)
		for i := range 100 {
			select {
			case <-ctx.Done():
				return
			case in <- i:
				if i == 30 {
					cancel()
					return
				}
			}
		}
	}()
	outs := shard.Repartition[int](10).Apply(ctx, in)
	assert.Len(t, outs, 10)
	seen := make([]bool, 100)
	wg := sync.WaitGroup{}
	for s := range 10 {
		wg.Go(func() {
			for v := range outs[s] {
				assert.False(t, seen[v])
				assert.LessOrEqual(t, 0, v)
				assert.LessOrEqual(t, v, 30)
				seen[v] = true
			}
		})
	}
	wg.Wait()
}

func TestRepartitionOneToOneEarlyCancelDuringRepartition(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		// send to in chan and then block from being able to send to out chan
		// then cancel
		in := make(chan int)
		sent := make(chan struct{})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		defer close(in)
		go func() {
			in <- 1
			close(sent)
		}()
		outs := shard.Repartition[int](1).WithBuffer(0).Apply(ctx, in)
		require.Len(t, outs, 1)
		out := outs[0]
		require.Equal(t, 0, cap(out))
		<-sent
		cancel()
		//time.Sleep(2 * time.Second)
		synctest.Wait()
		var got int
		for range out {
			got++
		}
		assert.Empty(t, got)
	})
}

func TestRepartitionOneToOneEarlyCancelWhileReceiving(t *testing.T) {
	// cancel just before sending
	in := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	outs := shard.Repartition[int](1).WithBuffer(0).Apply(ctx, in)
	cancel()
	require.Len(t, outs, 1)
	out := outs[0]
	require.Equal(t, 0, cap(out))
	var got int
	for v := range out {
		assert.Equal(t, 1, v)
		got++
	}
	assert.Equal(t, 0, got)
}

func generatorsForTestRepartitionOneToManyMultipleGeneratorsHardCancel(ctx context.Context) <-chan int {
	queue := make(chan int)
	jobs := 1000
	generators := 5
	var wgGen sync.WaitGroup
	for g := range generators {
		wgGen.Go(func() {
			for i := g; i < jobs; i += generators {
				select {
				case <-ctx.Done():
					return
				case queue <- i:
				}
			}
		})
	}

	go func() {
		wgGen.Wait()
		close(queue)
	}()
	return queue
}
func TestRepartitionOneToManyMultipleGeneratorsHardCancel(t *testing.T) {
	cutoff := 100
	in := make(chan int)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	queue := generatorsForTestRepartitionOneToManyMultipleGeneratorsHardCancel(ctx)

	// hard cutoff gate
	go func() {
		// deliberately NOT closing in chan
		// repartition should still successfully exit after cancel
		for v := range queue {
			if v >= cutoff {
				cancel()
				return
			}
			in <- v
		}
	}()

	outs := shard.Repartition[int](3).WithBuffer(0).Apply(ctx, in)
	require.Len(t, outs, 3)

	wg := sync.WaitGroup{}
	for _, out := range outs {
		wg.Go(func() {
			var got int
			for v := range out {
				assert.Less(t, v, cutoff)
				got++
			}
			assert.Less(t, got, cutoff)
		})
	}
	wg.Wait()
}
