package gather

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWorkersSpawnAndImmediatelyCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ws := &workerStation[int, int]{}
	ws.queue = make(chan job[int])
	ws.out = make(chan int, 1)

	go func() {
		defer close(ws.queue)
		select {
		case <-ctx.Done():
		case ws.queue <- job[int]{val: 1}:
		}
	}()

	ws.wgJob.Add(1)
	cancel()
	ws.StartWorker(ctx)
	assert.Empty(t, ws.out)
}

func TestReorder_CancelWhileBlockedOnOut(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		out := make(chan int)
		ordered := make(chan job[int])

		ws := &workerStation[int, int]{out: out, ordered: ordered}

		done := make(chan struct{})
		go func() { ws.Reorder(ctx); close(done) }()

		// send job to ordered chan but block from sending to out chan
		ordered <- job[int]{val: 42, index: 0}

		cancel()

		select {
		case <-done:
		case <-time.After(1 * time.Millisecond):
			t.Fatal("reorder did not exit on cancel")
		}
	})
}
