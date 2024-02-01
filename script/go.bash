#!/bin/bash
set -euxo pipefail
go run ./internal/cmd/buildtool gofixpath -- ./script/internal/go.bash "$@"
