# Gather

Gather is a lightweight channel-based concurrency library for Go.
It helps you build **worker pools, pipelines, and middleware**.

## Quick Example

```go
opts := []gather.Opt{
    gather.WithWorkerSize(100),
    gather.WithBufferSize(10),
    gather.WithOrderPreserved(),
}

handler := func(ctx context.Context, in Foo, _ *gather.Scope[Foo]) (Bar, error) {
    // do work...
    return Bar{}, nil
}

out := gather.Workers(ctx, in, handler, opts...) 

for v := range out {
    // consume
}
```

## Install

`go get github.com/???/gather`

## API at a glance

- Workers: start a worker pool that consumes an input channel and returns an output channel
- HandlerFunc: handles each job
- Scope: tools available to a handler (retries, safe go routines, etc)
- Middleware: wrap handlers and other middleware
- Chain: chains multiple middleware

## Design Philosophy

Gather provides the glue: workers, pipelines, and middleware.  
You design the concurrency patterns that fit your use case.

### Simple

- familiar middleware model (like "net/http")
- decouples middleware plumbing from business logic
- uses plain go primitives: context, channels, and functions

### Flexible

- leaves retry and error handling decisions to you
- lets you manage input/output channels directly
- no global state
- context-aware: honors cancellations, timeouts, and deadlines

## Should I use Gather?

Use Gather if channels are unavoidable
or if you need pipeline semantics that errgroup and sync.WaitGroup don't give you

### When Gather is a good fit

- you need middleware like retries, circuit breakers, backpressure, timeouts, log, etc
- multi-stage pipelines with fan-out/fan-in
- optional ordering with a reorder gate
- composability of stages

### When a simpler tool is better

- run independent tasks concurrently and stop on the first error -> use `errgroup`
- short-lived CPU-bound work -> use `sync.WaitGroup`
- background task waiting to close a channel -> use plain goroutine
- simple generator -> use plain goroutine

## Future Ideas

- sharding across multiple channels
- include `WithEventHook(hook func(Event))Opt` which can be used for logging/debugging
