package together

// Middleware - wraps handlers
type Middleware[IN any, OUT any] func(Handler[IN, OUT]) Handler[IN, OUT]

// Chain middleware together in FIFO execution order
func Chain[IN any, OUT any](mws ...Middleware[IN, OUT]) Middleware[IN, OUT] {
	return func(h Handler[IN, OUT]) Handler[IN, OUT] {
		for _, mw := range mws {
			h = mw(h)
		}
		return h
	}
}
