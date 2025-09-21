package gather_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"testing/synctest"
	"time"

	"gather"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkersSynchronous(t *testing.T) {
	ctx := context.Background()
	var got int
	for v := range gather.Workers(ctx, gen(ctx, 200), add(3)) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Equal(t, 200, got)
}

func TestWorkersSynchronousOrdered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		opts := []gather.Opt{
			gather.WithOrderPreserved(),
		}
		mw := mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second)
		var got int
		for v := range gather.Workers(ctx, gen(ctx, 1000), mw(add(3)), opts...) {
			require.Equal(t, got+3, v)
			got++
		}
		assert.Equal(t, 1000, got)
	})
}

func TestWorkersSynchronousOrderedWithEarlyCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mw := mwCancelOnInput[int, int](5, cancel)

	var got int
	for v := range gather.Workers(ctx, gen(ctx, 20), mw(add(3)), gather.WithOrderPreserved()) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Less(t, got, 3+5)
}

func TestPipelineSynchronous(t *testing.T) {
	ctx := context.Background()
	out1 := gather.Workers(ctx, gen(ctx, 20), add(3))
	out2 := gather.Workers(ctx, out1, subtract(3))
	var got int
	for v := range gather.Workers(ctx, out2, add(3)) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Equal(t, 20, got)
}

func TestPipelineSynchronousWithMultipleTypes(t *testing.T) {
	ctx := context.Background()
	out1 := gather.Workers(ctx, gen(ctx, 20), add(3))
	out2 := gather.Workers(ctx, out1, convert[int, float64]())
	out3 := gather.Workers(ctx, out2, subtract[float64](3))
	out4 := gather.Workers(ctx, out3, convert[float64, int]())
	var got int
	for v := range gather.Workers(context.Background(), out4, add(3)) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Equal(t, 20, got)
}

func TestPipelineSynchronousWithEarlyCancelAtLastStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// when input is 5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnInput[int, int](5, cancel)

	out1 := gather.Workers(ctx, gen(ctx, 20), add(3))
	out2 := gather.Workers(ctx, out1, subtract(3))
	var got int
	for v := range gather.Workers(ctx, out2, mw(add(3))) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Less(t, got, 3+5)
}

func TestPipelineSynchronousWithEarlyCancelAtFirstStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// when input is 5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnInput[int, int](5, cancel)

	out1 := gather.Workers(ctx, gen(ctx, 20), mw(add(3)))
	out2 := gather.Workers(ctx, out1, subtract(3))
	var got int
	for v := range gather.Workers(ctx, out2, add(3)) {
		require.Equal(t, got+3, v)
		got++
	}
	assert.Less(t, got, 3+5)
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

			opts := []gather.Opt{gather.WithWorkerSize(tt.workerSize)}
			if tt.bufferSize > 0 {
				opts = append(opts, gather.WithBufferSize(tt.bufferSize))
			}

			results := make([]int, 0, tt.jobs)
			for v := range gather.Workers(ctx, gen(ctx, tt.jobs), multiply(3), opts...) {
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

func TestWorkersOrdered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		opts := []gather.Opt{
			gather.WithWorkerSize(5),
			gather.WithBufferSize(5),
			gather.WithOrderPreserved(),
		}
		mw := mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second)
		var got int
		for v := range gather.Workers(ctx, gen(ctx, 1000), mw(add(3)), opts...) {
			require.Equal(t, got+3, v)
			got++
		}
		assert.Equal(t, 1000, got)
	})
}

func TestWorkersOrderedWithEarlyCancel(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		mw := gather.Chain(
			mwCancelOnInput[int, int](5, cancel),
			mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second),
		)
		opts := []gather.Opt{
			gather.WithWorkerSize(5),
			gather.WithBufferSize(5),
			gather.WithOrderPreserved(),
		}

		var got int
		for v := range gather.Workers(ctx, gen(ctx, 20), mw(add(3)), opts...) {
			require.Equal(t, got+3, v)
			got++
		}
		assert.Less(t, got, 3+5)
	})
}

func TestPipelineWithBackPressure(t *testing.T) {
	ctx := context.Background()
	out1 := gather.Workers(ctx, gen(ctx, 20), add(3), gather.WithWorkerSize(5))
	out2 := gather.Workers(ctx, out1, subtract(3), gather.WithWorkerSize(2))
	var got int
	for range gather.Workers(ctx, out2, add(3)) {
		got++
	}
	assert.Equal(t, 20, got)
}

func TestPipelineWithEarlyCancelAtLastStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// on input #5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnCount[int, int](5, cancel)

	out1 := gather.Workers(ctx, gen(ctx, 20), add(3), gather.WithWorkerSize(5))
	out2 := gather.Workers(ctx, out1, subtract(3), gather.WithWorkerSize(2))
	var got int
	for range gather.Workers(ctx, out2, mw(add(3))) {
		got++
	}
	assert.Less(t, got, 5)
}

func TestPipelineWithEarlyCancelAtFirstStage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// on input #5, it will cancel BEFORE processing the input
	// therefore, that result is not sent to out channel
	mw := mwCancelOnCount[int, int](5, cancel)

	out1 := gather.Workers(ctx, gen(ctx, 20), mw(add(3)), gather.WithWorkerSize(3))
	out2 := gather.Workers(ctx, out1, subtract(3), gather.WithWorkerSize(2))
	var got int
	for range gather.Workers(ctx, out2, add(3)) {
		got++
	}
	assert.LessOrEqual(t, got, 5)
}

func TestPipelineOrdered(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()

		opts := []gather.Opt{
			gather.WithWorkerSize(5),
			gather.WithBufferSize(5),
			gather.WithOrderPreserved(),
		}

		mw := mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second)

		out1 := gather.Workers(ctx, gen(ctx, 1000), mw(add(3)), opts...)
		out2 := gather.Workers(ctx, out1, subtract(3), opts...)
		var got int
		for v := range gather.Workers(ctx, out2, add(3), gather.WithOrderPreserved()) {
			require.Equal(t, got+3, v)
			got++
		}
		assert.Equal(t, 1000, got)
	})
}

func TestPipelineOrderedWithEarlyCancelAtFirstStage(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := []gather.Opt{
			gather.WithWorkerSize(5),
			gather.WithBufferSize(5),
			gather.WithOrderPreserved(),
		}

		// should cancel on input of 4
		mw := gather.Chain(
			mwCancelOnCount[int, int](5, cancel),
			mwRandomDelay[int, int](time.Now().UnixNano(), 0, time.Second),
		)

		out1 := gather.Workers(ctx, gen(ctx, 20), mw(add(3)), opts...)
		out2 := gather.Workers(ctx, out1, subtract(3), opts...)
		var got int
		for v := range gather.Workers(ctx, out2, add(3), gather.WithOrderPreserved()) {
			require.Equal(t, got+3, v)
			got++
		}
		assert.Less(t, got, 5)
	})
}
