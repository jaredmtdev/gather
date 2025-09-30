package seq_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/jaredmtdev/gather"
	"github.com/jaredmtdev/gather/internal/seq"
	"github.com/stretchr/testify/assert"
)

func TestWorkers(t *testing.T) {
	ctx := context.Background()
	workers := 3
	buffer := 1
	letters := strings.SplitSeq("abcdefghijklmnopqrstuvwxyz", "")
	var handler gather.HandlerFunc[string, string] = func(ctx context.Context, in string, _ *gather.Scope[string]) (string, error) {
		return strings.ToUpper(in), nil
	}
	out := seq.Workers(ctx, letters, handler, buffer, gather.WithWorkerSize(workers), gather.WithOrderPreserved())
	assert.Equal(t, "ABCDEFGHIJKLMNOPQRSTUVWXYZ", strings.Join(slices.Collect(out), ""))
}

func TestWorkersBreakEarly(t *testing.T) {
	ctx := context.Background()
	workers := 3
	buffer := 1
	letters := strings.SplitSeq("abcdefghijklmnopqrstuvwxyz", "")
	var handler gather.HandlerFunc[string, string] = func(ctx context.Context, in string, _ *gather.Scope[string]) (string, error) {
		return strings.ToUpper(in), nil
	}
	out := seq.Workers(ctx, letters, handler, buffer, gather.WithWorkerSize(workers), gather.WithOrderPreserved())
	sb := strings.Builder{}
	for v := range out {
		sb.WriteString(v)
		if v == "K" {
			break
		}
	}
	assert.Equal(t, "ABCDEFGHIJK", sb.String())
}
