package seq

import (
	"context"
	"gather"
	"gather/internal/shard"
	"iter"
)

// Shards - integrates shard.Apply using iter.Seq in place of channels
// by default, shards will use the same buffer as the input
func Shards[IN, OUT any](
	ctx context.Context,
	ins []iter.Seq[IN],
	handler gather.HandlerFunc[IN, OUT],
	buffer int,
	workerOpts ...gather.Opt,
) []iter.Seq[OUT] {
	opts := append([]gather.Opt{gather.WithBufferSize(buffer)}, workerOpts...)
	insCh := ToChans(ctx, ins, buffer)
	outsCh := shard.Apply(ctx, insCh, handler, opts...)
	return FromChans(ctx, outsCh)
}
