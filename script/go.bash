#!/bin/bash
set -euxo pipefail
# We invoke ./script/internal/go.bash through the gofixpath subcommand such that
# the "go" binary in PATH is the correct version of go.
#
# See https://github.com/ooni/probe/issues/2664
go run ./internal/cmd/buildtool gofixpath -- ./script/internal/go.bash "$@"
