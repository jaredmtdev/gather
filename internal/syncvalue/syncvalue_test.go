package syncvalue_test

import (
	"gather/internal/syncvalue"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentLoad(t *testing.T) {
	v := syncvalue.Value[int]{}
	v.Store(3)

	wg := sync.WaitGroup{}
	for range 1000 {
		wg.Go(func() {
			assert.Equal(t, 3, v.Load())
		})
	}
	wg.Wait()
}

func TestConcurrentStore(t *testing.T) {
	v := syncvalue.Value[int]{}
	wg := sync.WaitGroup{}
	for i := range 1000 {
		wg.Go(func() {
			v.Store(i)
		})
	}
	wg.Wait()

	assert.LessOrEqual(t, 0, v.Load())
	assert.Less(t, v.Load(), 1000)
}
