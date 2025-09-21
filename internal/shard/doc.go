// shard - package used to multiply stages into shards
// in most cases: just use together.Workers
// when you have a lot of jobs that complete instantly, sharding can help improve performance by breaking up into multiple channels
//
// this package is currently experimental
package shard
