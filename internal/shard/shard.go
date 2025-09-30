package shard

import (
	"context"

	"github.com/jaredmtdev/gather"
)

// Apply - applies sharding: each shard is a pool of gather.Workers
// best for large quantity of jobs that have near instant completion time
//
// shards can only guarantee order from within each shard (see workerOpts for ordering).
func Apply[IN, OUT any](
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
