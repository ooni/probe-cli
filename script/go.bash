#!/bin/bash
set -euxo pipefail

# We use ./pkg/gobash to ensure we execute the correct version of go.
#
# See https://github.com/ooni/probe/issues/2664 for context.

# Build the gobash.exe wrapper
(cd ./pkg/gobash && go build -v -o gobash.exe .)

# Download the exact version of Go we need
./pkg/gobash/gobash.exe download

# Make sure we're using the exact toolchain we've just downloaded
# See https://github.com/ooni/probe/issues/2695
export GOTOOLCHAIN=local

# Execute commands using such an exact version of go
./pkg/gobash/gobash.exe "$@"
