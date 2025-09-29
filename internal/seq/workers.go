package seq

import (
	"context"
	"github.com/jaredmtdev/gather"
	"iter"
)

// Workers - integrates with gather.Workers using iter.Seq in place of channels
// by default, the workers will use the same buffer as the input
func Workers[IN, OUT any](
	ctx context.Context,
	in iter.Seq[IN],
	handler gather.HandlerFunc[IN, OUT],
	buffer int,
	workerOpts ...gather.Opt,
) iter.Seq[OUT] {
	inCh := ToChan(ctx, in, buffer)
	opts := append([]gather.Opt{gather.WithBufferSize(buffer)}, workerOpts...)
	out := gather.Workers(ctx, inCh, handler, opts...)
	return FromChan(ctx, out)
}
