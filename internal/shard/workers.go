// shard - experimental package. may or may not release this for public use later
package shard

import (
	"context"
	"together"
)

// Workers - wraps around together.Workers by sharding multiple inputs/outputs
// each shard gets it's own worker pool
func Workers[IN, OUT any](
	ctx context.Context,
	ins []<-chan IN,
	handler together.HandlerFunc[IN, OUT],
	workerOpts ...together.Opt,
) []<-chan OUT {
	outs := make([]<-chan OUT, len(ins))

	// start each shard
	for i := range ins {
		outs[i] = together.Workers(ctx, ins[i], handler, workerOpts...)
	}

	return outs
}
