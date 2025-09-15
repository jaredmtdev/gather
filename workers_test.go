package together_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"together"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkersSynchronous(t *testing.T) {
	var i int
	for v := range together.Workers(context.Background(), gen(20), add(3)) {
		require.Equal(t, i+3, v)
		i++
	}
	assert.Equal(t, 20, i)
}

func TestPipelineSynchronous(t *testing.T) {
	ctx := context.Background()
	out1 := together.Workers(ctx, gen(20), add(3))
	out2 := together.Workers(ctx, out1, subtract(3))
	var i int
	for v := range together.Workers(ctx, out2, add(3)) {
		require.Equal(t, i+3, v)
		i++
	}
	assert.Equal(t, 20, i)
}

func TestPipelineSynchronousWithMultipleTypes(t *testing.T) {
	ctx := context.Background()
	out1 := together.Workers(ctx, gen(20), add(3))
	out2 := together.Workers(ctx, out1, convert[int, float64]())
	out3 := together.Workers(ctx, out2, subtract[float64](3))
	out4 := together.Workers(ctx, out3, convert[float64, int]())
	var i int
	for v := range together.Workers(context.Background(), out4, add(3)) {
		require.Equal(t, i+3, v)
		i++
	}
	assert.Equal(t, 20, i)
}

func TestPipelineSynchronousWithEarlyCancelAtLastStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// when input is 5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnInput[int, int](5, cancel)

	out1 := together.Workers(ctx, gen(20), add(3))
	out2 := together.Workers(ctx, out1, subtract(3))
	var i int
	for v := range together.Workers(ctx, out2, mw(add(3))) {
		require.Equal(t, i+3, v)
		i++
	}
	assert.Less(t, i, 3+5)
}

func TestPipelineSynchronousWithEarlyCancelAtFirstStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// when input is 5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnInput[int, int](5, cancel)

	out1 := together.Workers(ctx, gen(20), mw(add(3)))
	out2 := together.Workers(ctx, out1, subtract(3))
	var i int
	for v := range together.Workers(ctx, out2, add(3)) {
		require.Equal(t, i+3, v)
		i++
	}
	assert.Less(t, i, 3+5)
}

func TestMultipleWorkers(t *testing.T) {
	tests := []struct {
		jobs       int
		workerSize int
		bufferSize int
	}{
		{jobs: 20, workerSize: 5, bufferSize: 0},
		{jobs: 20, workerSize: 5, bufferSize: 3},
		{jobs: 20, workerSize: 5, bufferSize: 3},
		{jobs: 1000, workerSize: 8, bufferSize: 0},
		{jobs: 1000, workerSize: 1, bufferSize: 8},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("jobs=%v, workerSize=%v, bufferSize=%v", tt.jobs, tt.workerSize, tt.bufferSize), func(t *testing.T) {
			ctx := context.Background()

			opts := []together.Opt{together.WithWorkerSize(tt.workerSize)}
			if tt.bufferSize > 0 {
				opts = append(opts, together.WithBufferSize(tt.bufferSize))
			}

			results := make([]int, 0, tt.jobs)
			for v := range together.Workers(ctx, gen(tt.jobs), multiply(3), opts...) {
				results = append(results, v)
			}

			require.Len(t, results, tt.jobs)

			slices.Sort(results)
			for i, v := range results {
				require.Equal(t, i*3, v)
			}
			assert.Equal(t, (tt.jobs-1)*3, results[tt.jobs-1])
		})
	}
}

func TestPipelineWithBackPressure(t *testing.T) {
	ctx := context.Background()
	out1 := together.Workers(ctx, gen(20), add(3), together.WithWorkerSize(5))
	out2 := together.Workers(ctx, out1, subtract(3), together.WithWorkerSize(2))
	var i int
	for range together.Workers(ctx, out2, add(3)) {
		i++
	}
	assert.Equal(t, 20, i)
}

func TestPipelineWithEarlyCancelAtLastStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// on input #5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnCount[int, int](5, cancel)

	out1 := together.Workers(ctx, gen(20), add(3), together.WithWorkerSize(5))
	out2 := together.Workers(ctx, out1, subtract(3), together.WithWorkerSize(2))
	var i int
	for range together.Workers(ctx, out2, mw(add(3))) {
		i++
	}
	assert.Less(t, i, 5)
}

func TestPipelineWithEarlyCancelAtFirstStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// on input #5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnCount[int, int](5, cancel)

	out1 := together.Workers(ctx, gen(20), mw(add(3)), together.WithWorkerSize(3))
	out2 := together.Workers(ctx, out1, subtract(3), together.WithWorkerSize(2))
	var i int
	for range together.Workers(ctx, out2, add(3)) {
		i++
	}
	assert.Less(t, i, 5)
}
