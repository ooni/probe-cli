#!/bin/sh
set -euxo pipefail

# required because the container is running as root
git config --global --add safe.directory /jafar

# TODO(bassosimone): investigate why using CGO_ENABLED=1 is such
# that all DNS lookups return `dns_nxdomain_error`
export CGO_ENABLED=0

# TODO(bassosimone): because this script runs as root, it's not
# possible to save the caching directories in github actions but
# doing that would making re-executing these scripts faster.
export GOPATH=/jafar/QA/GOPATH
export GOCACHE=/jafar/QA/GOCACHE

go build -v ./internal/cmd/miniooni

go build -v ./internal/cmd/jafar

sudo ./QA/$1.py ./miniooni
