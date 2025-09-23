package gather

import (
	"context"
	"testing"
	"time"
)

func TestReorder_CancelWhileBlockedOnOut(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	out := make(chan int)
	ordered := make(chan job[int], 1)

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
}
