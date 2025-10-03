package gather

import (
	"context"
	"slices"
)

// HandlerFunc - function used to handle a single request sent to a worker
// Error handling is done here. user can:
//   - cancel the context if needed for immediate shutdown
//   - for graceful shutdown: user controls generator. can just close in chan and then let all downstream stages finish
//   - send to their own err channel (which could be processed by another Workers)
//   - use scope for retries, ReplyTo pattern, etc
type HandlerFunc[IN any, OUT any] func(ctx context.Context, in IN, scope *Scope[IN]) (OUT, error)

// Middleware wraps a HandlerFunc and returns a new HandlerFunc.
// The returned handler may run logic before and/or after invoking next,
// or choose not to call next (short-circuit).
//
// Middleware is composable: multiple middlewares can be applied in order.
type Middleware[IN any, OUT any] func(next HandlerFunc[IN, OUT]) HandlerFunc[IN, OUT]

// Chain middleware together in FIFO execution order.
//
// Chain(m1,m2)(h) == m1(m2(h))
func Chain[IN any, OUT any](mws ...Middleware[IN, OUT]) Middleware[IN, OUT] {
	return func(h HandlerFunc[IN, OUT]) HandlerFunc[IN, OUT] {
		for _, mw := range slices.Backward(mws) {
			h = mw(h)
		}
		return h
	}
}
