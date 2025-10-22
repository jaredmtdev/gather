  <div align="center"><pre>
  _____       _______ _    _ ______ _____  
 / ____|   /\|__   __| |  | |  ____|  __ \ 
| |  __   /  \  | |  | |__| | |__  | |__) |
| | |_ | / /\ \ | |  |  __  |  __| |  _  / 
| |__| |/ ____ \| |  | |  | | |____| | \ \ 
 \_____/_/    \_\_|  |_|  |_|______|_|  \_\
 </pre></div>
 <div align="center"><img src="gatherlogo.jpg" alt="gather logo" width="200" /></div>

<div align="center">

 [![Go Report Card](https://goreportcard.com/badge/github.com/jaredmtdev/gather)](https://goreportcard.com/report/github.com/jaredmtdev/gather)
 [![OpenSSF Score](https://api.scorecard.dev/projects/github.com/jaredmtdev/gather/badge)](https://scorecard.dev/viewer/?uri=github.com/jaredmtdev/gather)
 [![Test Status](https://img.shields.io/github/actions/workflow/status/jaredmtdev/gather/test.yml?branch=main&logo=GitHub&label=test)](https://github.com/jaredmtdev/gather/actions/workflows/test.yml?query=branch%3Amain)
 [![Codecov](https://img.shields.io/codecov/c/github/jaredmtdev/gather?logo=codecov)](https://codecov.io/gh/jaredmtdev/gather)
 [![GitHub Release](https://img.shields.io/github/v/release/jaredmtdev/gather?label=version)](https://github.com/jaredmtdev/gather/releases)
 [![Go Reference](https://pkg.go.dev/badge/github.com/jaredmtdev/gather.svg)](https://pkg.go.dev/github.com/jaredmtdev/gather)

</div>

Gather is a lightweight channel-based concurrency library for Go.
It helps you build **worker pools, pipelines, and middleware**.

## Quick Example

```go
opts := []gather.Opt {
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

## Add to your project

```
go get github.com/jaredmtdev/gather
```

## Stability

This project is currently pre-v1.0. The API may change between **minor** versions.
We will guarantee SemVer versioning starting from `v1.0.0`.

## API at a glance

- Workers: start a worker pool that consumes an input channel and returns an output channel
- HandlerFunc: handles each job
- Scope: tools available to a handler (i.e. retries)
- Middleware: wrap handlers and other middleware
- Chain: chains multiple middleware

More details [here](https://pkg.go.dev/github.com/jaredmtdev/gather).

## Design Philosophy

Gather provides the glue: worker pools, pipelines, and a `Middleware` type.
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

- independent tasks, fail fast -> use `errgroup`
- 100% CPU-bound work -> use `sync.WaitGroup`
- jobs that complete instantly
  - simplicity / minimal memory -> use `iter` package
  - best execution time
    - can avoid channels -> use fixed workers processing batches (fastest)
    - want decoupling/backpressure/middleware -> use `gather` with multiple workers + batches
      - large enough batches ammortize channel cost to almost nothing
- background task waiting to close a channel -> use plain goroutine
- simple generator -> use plain goroutine or `iter.Seq`

## Getting Started

### Building a worker pool

#### 1: Build your handler

This handles each item sent to the worker pool

```go
handler := func(ctx context.Context, in Foo, scope *gather.Scope[Foo]) (Bar, error) {
    // do work...
    return Bar{}, nil
}
```

The `ctx` parameter can be cancelled at any time to shut down the worker pool.
Note that it's important for the handler to also honor any context cancellation for a quicker cancellation.

The `in` parameter can be any type and the response from the workerpool can be any type.
When returning an error, the result is not sent to the output channel.
If needed, the error can also be sent to an error channel (which you create) and then processed in a separate goroutine (which you define).
The error response also comes in handy when building middleware.

Optionally, make custom middleware to conveniently wrap around the handler:

```go
wrappedHandler := logger(retries(rateLimiter(handler)))

// alternatively, chain the middleware with Chain:
mw := gather.Chain(rateLimiter, retries, logger)
wrappedHandler := mw(handler)
```

See [examples/internal/samplemiddleware/](/examples/internal/samplemiddleware/samplemiddleware.go) for more detailed examples on building middleware.

The `scope` parameter provides extra capabilities to the handler such as retries.

#### 2: Build your generator

You need to have a channel of any type and send data to it. Example:

```go
in := make(chan int)
go func(){
  defer close(in)
  for i := range 100 {
      select{
          case <-ctx.Done():
            return
          case in <- i:
      }
  }
}
```

#### 3: Configure and run the worker pool

```go
opts = []gather.Opt{
    gather.WithWorkerSize(runtime.GOMAXPROCS(0)),
    gather.WithBufferSize(1),
}
out := gather.Workers(ctx, in, handler, opts...) 
```

This will return an output channel.
The channel must be consumed or drained so the workers don't get blocked:

```go
// consume output
for v := range out {
    fmt.Println(v)
}

// alternatively, drain output if it doesn't need to be consumed
for range out{
}
```

Notice the `opts...`. These options are used to configure the worker pool.
Look for any function starting with `gather.With`.
Here you can configure things like the number of workers, channel buffer size, preserve order, etc.

### Building a pipeline

The above worker pools are a single stage of the pipeline.
To build a pipeline, just build multiple worker pools and pass the output of one into the next:

```go
out1 := gather.Workers(ctx, in, handler1, opts...) 
out2 := gather.Workers(ctx, out1, handler2, opts...) 
out3 := gather.Workers(ctx, out2, handler3, opts...) 
for range out3 {
    // drain
}
```

This is quite simple and gives full control!
You can tune the configuration of each stage of the pipeline.
You could cancel at any stage to stop the entire pipeline.

### Examples

Please see [examples/](/examples/) folder for some simple examples.

## Future Ideas

- sharding across multiple channels
- seq package: offer synchronous helpers that utilize iter.Seq and integrate nicely with Gather
- `WithElasticWorkers(minWorkerSize,TTL)` allow workers to scale up/down as needed to save on memory and overhead.
- `WithControls(*gather.Controls)` provide a controller to tune the config at runtime and collect stat snapshots.
- include `WithEventHook(hook func(Event)) Opt` which can be used for logging/debugging
