
# together

A lightweight concurrency library built for simplicity and control.

## Enables you to build

- worker pools
- pipelines
- middleware

## How it is simple

Following go's philosphy, I don't want this library to become bloated with features that are not needed.
This library is simple because it:

- enables developers to build just about any concurrency model
- provides a middleware mechanism
  - very similar to known middleware patterns (like http package)
  - decouples middleware from business logic - making it easier to read and debug

## How it gives control

The goal of this library is to provide the glue that ties all your concurrency patterns together.

By using this library, you are organizing your concurrency patterns into smaller more manageable parts.

However, you are still responsible to build your middleware.
It is up to YOU to determine how you want your retry logic to work.
Why? Because there is no one-size-fits-all solution.

You are responsible for providing the input channel and for managing the output channel.
Why? Because, there are infinite concurrency patterns.
This library will not limit the use cases by favoring any particular pattern

## In the works

in the future, this library may provide these features:
- sharding: using multiple channels to help relieve some pressure where a buffer is not enough
