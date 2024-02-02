#!/bin/bash
set -euxo pipefail

# We use ./pkg/gobash to ensure we execute the correct version of go.
#
# See https://github.com/ooni/probe/issues/2664 for context.
(cd ./pkg/gobash && go build -v -o gobash.exe .)
./pkg/gobash/gobash.exe download
./pkg/gobash/gobash.exe "$@"
