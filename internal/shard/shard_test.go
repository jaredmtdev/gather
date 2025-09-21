package shard_test

import (
	"context"
	"gather"
	"gather/internal/shard"
	"testing"

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
