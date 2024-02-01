#!/bin/bash
set -euxo pipefail
# We invoke ./script/internal/go.bash through the gofixpath subcommand such that
# the "go" binary in PATH is the correct version of go.
go run ./internal/cmd/buildtool gofixpath -- ./script/internal/go.bash "$@"
