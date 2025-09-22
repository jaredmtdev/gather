# together

Together is a lightweight channel-based concurrency library for Go.
It helps you build **worker pools, pipelines, and middleware**.

## Quick Example

## Install

`go get github.com/???/together`

## API at a glance

- Workers: function to start a worker pool
- HandlerFunc: function to handle each request sent to worker pool
- Scope: provided to HandlerFunc to enable Retries, safe go routines, etc
- Middleware: helper to allow you to wrap middleware around the handler
- Chain: helper to more cleanly chain middleware together

## Design Philosophy

Together provides the glue: the tools for workers, pipelines, and middleware.  
You design the concurrency patterns that fit your use case.

Together is simple because it:

- enables developers to build almost any concurrency model
- provides familiar middleware mechanisms (like "net/http")
- decouples middleware logic from business logic for easier testing and debugging

Together is flexible because it:

- leaves retry and error handling decisions to you
- lets you manage input/ouput channels directly
- avoids bias or enforcement of any particular pattern

## Future Ideas

- sharding across multiple channels
