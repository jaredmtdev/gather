#!/bin/bash -ex
#
# Build one fuzz target binary per FuzzXxx entrypoint using compile_go_fuzzer.
# Args to compile_go_fuzzer:
#   <pkg-path> <fuzz-func> <out-binary-name> [optional-build-tag]

go mod download

compile_go_fuzzer github.com/jaredmtdev/gather/internal/fuzzgather FuzzScopeRetryAfterWhenNoError fuzz_scope_retry clusterfuzzlite
