package together

/*
note: we do need to require something that takes a func as a parameter.
that's the only way to be able to do something before and after
*/

type Middleware[IN any, OUT any] func(Handler[IN, OUT]) Handler[IN, OUT]

// FIFO execution order
func Chain[IN any, OUT any](mws ...Middleware[IN, OUT]) Middleware[IN, OUT] {
	return func(h Handler[IN, OUT]) Handler[IN, OUT] {
		for _, mw := range mws {
			h = mw(h)
		}
		return h
	}
}

// maybe start here? could build from the lowest level up

// TODO: provide MiddlewareChain()

// type MiddlewareFunc[T any] func() (T, error)
//
// func (m MiddlewareFunc[T]) Do(f MiddlewareFunc[T]) (T, error) {
// 	return m()
// }
//
// type Middleware[T any] func(MiddlewareFunc[T]) MiddlewareFunc[T]
//
// func MiddlewareChain2[T any](f MiddlewareFunc[T], m ...MiddlewareFunc[T]) MiddlewareFunc[T] {
// 	wrapper := f
// 	for i := range slices.Backward(m) {
// 		next := wrapper
// 		wrapper = func() (T, error) {
// 			return m[i]()
// 		}
// 	}
// }
//
// func MiddlewareChain[T any](f MiddlewareFunc[T], m ...Middleware[T]) MiddlewareFunc[T] {
// 	for i := range m {
// 		f = m[i](f)
// 	}
// 	return f
// }

// // this approach is similar to http package but we probably don't need this much complexity
// // this may make sense if I decide to change HandlerFunc to take an input but I don't think that would be necessary
// type MiddlewareHandler[T any] interface {
// 	Do() (T, error)
// }
// type Middleware[T any] func(MiddlewareHandler[T]) MiddlewareHandler[T]
// type MiddlewareHandlerFunc[T any] func() (T, error)
// func (m MiddlewareHandlerFunc[T]) Do() (T, error) {
// 	return m()
// }
// func MiddlewareChain[T any](h MiddlewareHandler[T], m ...Middleware[T]) MiddlewareHandler[T] {
// 	for i := range m {
// 		h = m[i](h)
// 	}
// 	return h
// }

// is there an easier implementation with middleware as a func instead of an interface??
// otherwise, go back to what I had

// type Middleware[T any] interface {
// 	Do(fn func() (T, error)) (T, error)
// }
//
// type middlewareClient[T any] struct {
// 	middlewares []Middleware[T]
// 	//middleware  Middleware[T]
// }
//
// func MiddlewareChain[T any](middlewares ...Middleware[T]) Middleware[T] {
// 	return &middlewareClient[T]{
// 		middlewares: middlewares,
// 	}
// }
//
// func (c *middlewareClient[T]) Do(fn func() (T, error)) (T, error) {
// 	wrapper := fn
// 	for _, mw := range slices.Backward(c.middlewares) {
// 		next := wrapper
// 		wrapper = func() (T, error) {
// 			return mw.Do(next)
// 		}
// 	}
// 	return wrapper()
// }
