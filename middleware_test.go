package gather_test

import (
	"context"
	"testing"

	"github.com/jaredmtdev/gather"
	"github.com/stretchr/testify/assert"
)

func TestChainOrderFIFO(t *testing.T) {
	ctx := context.Background()
	jobs := 100
	order := make(chan int, jobs*2)
	mw1 := mwId[int, int](0, order)
	mw2 := mwId[int, int](1, order)
	mw := gather.Chain(mw1, mw2)
	for range gather.Workers(ctx, gen(ctx, jobs), mw(add(1))) {
	}
	close(order)
	i := 0
	for v := range order {
		assert.Equal(t, i%2, v)
		i++
	}
}

func TestManualChainOrderFIFO(t *testing.T) {
	ctx := context.Background()
	jobs := 100
	order := make(chan int, jobs*2)
	mw1 := mwId[int, int](0, order)
	mw2 := mwId[int, int](1, order)
	for range gather.Workers(ctx, gen(ctx, jobs), mw1(mw2((add(1))))) {
	}
	close(order)
	i := 0
	for v := range order {
		assert.Equal(t, i%2, v)
		i++
	}
}
