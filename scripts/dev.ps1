#!/usr/bin/env pwsh

$ErrorActionPreference = "Stop"

# absolute path
$HostPath = (Get-Location).Path

docker run -it --rm `
  -v "${HostPath}:/gather" `
  -w /gather `
  golang:1.25.1@sha256:d7098379b7da665ab25b99795465ec320b1ca9d4addb9f77409c4827dc904211 `
  bash -c 'go mod tidy && bash'
