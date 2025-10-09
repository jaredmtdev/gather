#!/bin/bash -ex

go mod download

compile_go_fuzzer github.com/jaredmtdev/gather/internal/fuzzgather FuzzScopeRetryAfterWhenNoError fuzz_scope_retry clusterfuzzlite
