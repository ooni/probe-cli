#!/bin/bash

# Computes coverage inside an environment where we unshared the network namespace
# to ensure unit tests don't depend on the network.

set -euxo pipefail

# obtain the full path of the go executable
go=$(which go)

# populate the vendor directory so we don't need the network in `go test`
go mod vendor

# run tests using a different network namespace
sudo unshare --net $go test -short -race -coverprofile=probe-cli.cov ./...

