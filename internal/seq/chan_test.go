package seq_test

import (
	"context"
	"iter"
	"sync"
	"testing"
	"testing/synctest"

	"github.com/jaredmtdev/gather/internal/seq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToChan(t *testing.T) {
	ctx := context.Background()
	in := func() iter.Seq[int] {
		return func(yield func(v int) bool) {
			for i := range 10 {
				if !yield(i) {
					return
				}
			}
		}
	}

	out := seq.ToChan(ctx, in(), 3)
	assert.Equal(t, 3, cap(out))
	var got int
	for v := range out {
		require.Equal(t, got, v)
		got++
	}
	assert.Equal(t, 10, got)
}

func TestToChanBreakEarly(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		in := func() iter.Seq[int] {
			return func(yield func(v int) bool) {
				for i := range 10 {
					if !yield(i) {
						return
					}
				}
			}
		}

		out := seq.ToChan(ctx, in(), 0)
		assert.Equal(t, 0, cap(out))
		var got int
		for v := range out {
			require.Equal(t, got, v)
			got++
			if got == 5 {
				cancel()
				synctest.Wait()
			}
		}

		assert.Equal(t, 5, got)
	})
}

func TestToChans(t *testing.T) {
	ctx := context.Background()
	in := func() iter.Seq[int] {
		return func(yield func(v int) bool) {
			for i := range 10 {
				if !yield(i) {
					return
				}
			}
		}
	}
	ins := make([]iter.Seq[int], 10)
	for i := range ins {
		ins[i] = in()
	}

	outs := seq.ToChans(ctx, ins, 3)
	assert.Len(t, outs, 10)

	assert.Equal(t, 3, cap(outs[0]))
	for _, out := range outs {
		var got int
		for v := range out {
			require.Equal(t, got, v)
			got++
		}
		assert.Equal(t, 10, got)
	}
}

func TestToChansBreakEarly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var in iter.Seq[int] = func(yield func(v int) bool) {
		for i := range 10 {
			if !yield(i) {
				return
			}
		}
	}
	ins := make([]iter.Seq[int], 10)
	for i := range ins {
		ins[i] = in
	}

	outs := seq.ToChans(ctx, ins, 0)
	assert.Len(t, outs, 10)

	assert.Equal(t, 0, cap(outs[0]))
	wg := sync.WaitGroup{}
	for out := range outs {
		wg.Go(func() {
			var got int
			for v := range out {
				require.Equal(t, got, v)
				got++
				if got >= 5 {
					cancel()
					break
				}
			}
			assert.LessOrEqual(t, got, 5)
		})
	}
	wg.Wait()
}

func TestFromChan(t *testing.T) {
	ctx := context.Background()
	in := make(chan int)
	go func() {
		defer close(in)
		for i := range 20 {
			select {
			case <-ctx.Done():
				return
			case in <- i:
			}
		}
	}()
	inSeq := seq.FromChan(ctx, in)
	wg := sync.WaitGroup{}
	seen := make([]bool, 20)
	for range 3 {
		wg.Go(func() {
			for v := range inSeq {
				assert.False(t, seen[v])
				seen[v] = true
			}
		})
	}
	wg.Wait()
	for _, v := range seen {
		assert.True(t, v)
	}
}

func TestFromChanBreakEarly(t *testing.T) {
	ctx := context.Background()
	in := make(chan int)
	go func() {
		defer close(in)
		for i := range 20 {
			in <- i
		}
	}()
	inSeq := seq.FromChan(ctx, in)
	wg := sync.WaitGroup{}
	seen := make([]bool, 20)
	for range 3 {
		wg.Go(func() {
			for v := range inSeq {
				assert.False(t, seen[v])
				seen[v] = true
				if v >= 10 {
					return
				}
			}
		})
	}
	wg.Wait()
	for i, v := range seen {
		if i < 10+3 {
			assert.True(t, v)
		} else {
			assert.False(t, v)
		}
	}
}

func TestFromChanEarlyCancelFromGenerator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	in := make(chan int)
	defer close(in)
	go func() {
		for i := range 10 {
			in <- i
		}
		cancel()
	}()
	inSeq := seq.FromChan(ctx, in)
	wg := sync.WaitGroup{}
	seen := make([]bool, 20)
	for range 3 {
		wg.Go(func() {
			for v := range inSeq {
				assert.False(t, seen[v])
				seen[v] = true
			}
		})
	}
	wg.Wait()
	for i, v := range seen {
		if i < 10 {
			assert.True(t, v)
		} else {
			assert.False(t, v)
		}
	}
}

func assertOutputOnTestFromChans(t *testing.T, s int, insSeq []iter.Seq[int]) {
	seen := make([]bool, 20)
	wgInner := sync.WaitGroup{}
	for range 3 {
		wgInner.Go(func() {
			for v := range insSeq[s] {
				assert.False(t, seen[v])
				seen[v] = true
			}
		})
	}
	wgInner.Wait()
	for i := range 20 {
		require.True(t, seen[i])
	}
}

func TestFromChans(t *testing.T) {
	ctx := context.Background()
	ins := make([]<-chan int, 5)
	wg := sync.WaitGroup{}
	wg.Add(5)
	for w := range 5 {
		go func() {
			in := make(chan int)
			ins[w] = in
			wg.Done()
			defer close(in)
			for i := range 20 {
				select {
				case <-ctx.Done():
					return
				case in <- i:
				}
			}
		}()
	}
	wg.Wait()
	insSeq := seq.FromChans(ctx, ins)
	require.Len(t, insSeq, 5)

	// each seq should be able to be processed by multiple workers
	for s := range 5 {
		wg.Go(func() {
			assertOutputOnTestFromChans(t, s, insSeq)
		})
	}
	wg.Wait()
}

func assertOutputOnTestFromChansBreakEarly(t *testing.T, s int, insSeq []iter.Seq[int]) {
	seen := make([]bool, 20)
	wgInner := sync.WaitGroup{}
	for range 3 {
		wgInner.Go(func() {
			for v := range insSeq[s] {
				assert.False(t, seen[v])
				seen[v] = true
				if v >= 10 {
					break
				}
			}
		})
	}
	wgInner.Wait()
	for i := range 20 {
		if i < 10+3 {
			require.True(t, seen[i])
		} else {
			require.False(t, seen[i])
		}
	}
}

func TestFromChansBreakEarly(t *testing.T) {
	ctx := context.Background()
	ins := make([]<-chan int, 5)
	wg := sync.WaitGroup{}
	wg.Add(5)
	for w := range 5 {
		go func() {
			in := make(chan int)
			ins[w] = in
			wg.Done()
			defer close(in)
			for i := range 20 {
				select {
				case <-ctx.Done():
					return
				case in <- i:
				}
			}
		}()
	}
	wg.Wait()
	insSeq := seq.FromChans(ctx, ins)
	require.Len(t, insSeq, 5)

	// each seq should be able to be processed by multiple workers
	for s := range 5 {
		wg.Go(func() {
			assertOutputOnTestFromChansBreakEarly(t, s, insSeq)
		})
	}
	wg.Wait()
}

func assertOutputOnTestFromChansCancelEarlyFromGenerator(t *testing.T, s int, insSeq []iter.Seq[int]) {
	seen := make([]bool, 20)
	wgInner := sync.WaitGroup{}
	for range 3 {
		wgInner.Go(func() {
			for v := range insSeq[s] {
				assert.False(t, seen[v])
				seen[v] = true
			}
		})
	}
	wgInner.Wait()
	var trueCount int
	for i := range 20 {
		if i >= 10 {
			assert.False(t, seen[i], i)
		}
		if seen[i] {
			trueCount++
		}
	}
	assert.LessOrEqual(t, trueCount, 10)
}

func TestFromChansCancelEarlyFromGenerator(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ins := make([]<-chan int, 5)
	wg := sync.WaitGroup{}
	wg.Add(5)
	for w := range 5 {
		go func() {
			in := make(chan int)
			ins[w] = in
			wg.Done()
			defer close(in)
			for i := range 20 {
				if i >= 10 {
					cancel()
				}
				select {
				case <-ctx.Done():
					return
				case in <- i:
				}
			}
		}()
	}
	wg.Wait()
	insSeq := seq.FromChans(ctx, ins)
	require.Len(t, insSeq, 5)

	// each seq should be able to be processed by multiple workers
	for s := range 5 {
		wg.Go(func() {
			assertOutputOnTestFromChansCancelEarlyFromGenerator(t, s, insSeq)
		})
	}
	wg.Wait()
}
