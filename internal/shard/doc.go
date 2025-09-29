// Package shard - package used to multiply stages into shards
// in most cases: just use gather.Workers
//
// some situations where sharding might make sense:
// - when you have a lot of jobs that complete instantly, sharding can help improve performance by breaking up into multiple channels
// - you want to split the work by keys (each key gets it's own ordering or rate limiting etc)
// - when a stage multiplies data - sharding can help contain an occasional large data multiplication to a single key (and not block progress on other keys)
//
// this package is currently experimental
package shard
