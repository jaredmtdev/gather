package seq_test

import (
	"context"
	"gather"
	"gather/internal/seq"
	"iter"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShards(t *testing.T) {
	ctx := context.Background()
	buffer := 1
	workers := 3
	ins := make([]iter.Seq[string], 10)
	for i := range 10 {
		ins[i] = strings.SplitSeq("abcdefghijklmnopqrstuvwxyz", "")
	}
	var handler gather.HandlerFunc[string, string] = func(ctx context.Context, in string, _ *gather.Scope[string]) (string, error) {
		return strings.ToUpper(in), nil
	}
	outs := seq.Shards(ctx, ins, handler, buffer, gather.WithWorkerSize(workers), gather.WithOrderPreserved())
	require.Len(t, outs, 10)
	wg := sync.WaitGroup{}
	for i, out := range outs {
		wg.Go(func() {
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			seen := make([]bool, 26)
			wgInner := sync.WaitGroup{}
			for range workers {
				wgInner.Go(func() {
					for v := range out {
						assert.Contains(t, letters, v, v, i)
						ind := int(v[0] - 'A')
						assert.False(t, seen[ind])
						seen[ind] = true
					}
				})
			}
			wgInner.Wait()
			for i := range seen {
				assert.True(t, seen[i], i)
			}
		})
	}
	wg.Wait()
}

func TestShardsEarlyBreak(t *testing.T) {
	ctx := context.Background()
	buffer := 1
	workers := 3
	ins := make([]iter.Seq[string], 10)
	for i := range 10 {
		ins[i] = strings.SplitSeq("abcdefghijklmnopqrstuvwxyz", "")
	}
	var handler gather.HandlerFunc[string, string] = func(ctx context.Context, in string, _ *gather.Scope[string]) (string, error) {
		return strings.ToUpper(in), nil
	}
	outs := seq.Shards(ctx, ins, handler, buffer, gather.WithWorkerSize(workers), gather.WithOrderPreserved())
	require.Len(t, outs, 10)
	wg := sync.WaitGroup{}
	for i, out := range outs {
		wg.Go(func() {
			letters := "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
			seen := make([]bool, 26)
			wgInner := sync.WaitGroup{}
			for range workers {
				wgInner.Go(func() {
					for v := range out {
						assert.Contains(t, letters, v, v, i)
						ind := int(v[0] - 'A')
						assert.False(t, seen[ind])
						seen[ind] = true
						if ind >= 10 {
							break
						}
					}
				})
			}
			wgInner.Wait()
			for i := range seen {
				if i < 10+3 {
					assert.True(t, seen[i], i)
				} else {
					assert.False(t, seen[i], i)
				}
			}
		})
	}
	wg.Wait()
}
