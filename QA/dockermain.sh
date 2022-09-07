#!/bin/sh
set -euxo pipefail

# required because the container is running as root
git config --global --add safe.directory /jafar

# TODO(bassosimone): investigate why using CGO_ENABLED=1 is such
# that all DNS lookups return `dns_nxdomain_error`
export CGO_ENABLED=0

export GOPATH=/jafar/QA/GOPATH
export GOCACHE=/jafar/QA/GOCACHE

go build -v ./internal/cmd/miniooni

go build -v ./internal/cmd/jafar

sudo ./QA/$1.py ./miniooni
