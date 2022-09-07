#!/bin/sh
set -ex
# TODO(bassosimone): investigate why using CGO_ENABLED=1 is such
# that all DNS lookups return `dns_nxdomain_error`
export CGO_ENABLED=0 GOPATH=/jafar/QA/GOPATH GOCACHE=/jafar/QA/GOCACHE
go build -v ./internal/cmd/miniooni
go build -v ./internal/cmd/jafar
sudo ./QA/$1.py ./miniooni
