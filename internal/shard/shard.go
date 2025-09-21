package shard

import (
	"context"
	"gather"
)

// Shards - wraps around gather.Workers by sharding multiple inputs/outputs
// each shard spawns it's own worker pool
func Shards[IN, OUT any](
	ctx context.Context,
	ins []<-chan IN,
	handler gather.HandlerFunc[IN, OUT],
	workerOpts ...gather.Opt,
) []<-chan OUT {
	outs := make([]<-chan OUT, len(ins))

	// start each shard
	for i := range ins {
		outs[i] = gather.Workers(ctx, ins[i], handler, workerOpts...)
	}

	return outs
}
