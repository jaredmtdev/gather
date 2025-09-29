package op_test

import (
	"github.com/jaredmtdev/gather/internal/op"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPosMod(t *testing.T) {
	for i := -10; i < 20; i++ {
		result := op.PosMod(i, 5)
		assert.LessOrEqual(t, 0, result)
		assert.LessOrEqual(t, result, 5)
	}
	assert.Equal(t, 2, op.PosMod(2, 3))
	assert.Equal(t, 0, op.PosMod(3, 3))
	assert.Equal(t, 0, op.PosMod(-3, 3))
	assert.Equal(t, 2, op.PosMod(-1, 3))
}
