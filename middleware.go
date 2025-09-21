package gather

import "context"

// HandlerFunc - function used to handle a single request sent to the worker
// error handling done here. user can:
//   - cancel the context if needed for immediate shutdown
//   - for graceful shutdown: user controls generator. can just close in chan and then let all downstream stages finish
//   - send to their own err channel (which could be processed by another Workers)
//   - use workerHandler for retries, ReplyTo pattern, etc
type HandlerFunc[IN any, OUT any] func(ctx context.Context, in IN, scope *Scope[IN]) (OUT, error)

// Middleware - wraps handlers.
type Middleware[IN any, OUT any] func(HandlerFunc[IN, OUT]) HandlerFunc[IN, OUT]

// Chain - middleware gather in FIFO execution order.
func Chain[IN any, OUT any](mws ...Middleware[IN, OUT]) Middleware[IN, OUT] {
	return func(h HandlerFunc[IN, OUT]) HandlerFunc[IN, OUT] {
		for _, mw := range mws {
			h = mw(h)
		}
		return h
	}
}
