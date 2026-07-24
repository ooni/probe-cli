#!/bin/bash

#
# Script invoked by ./script/linuxcoverage.bash to run tests with coverage
# using a separate network namespace with only loopback support.
#
# The first an unique argument is the path to the go binary to use.
#

set -euxo pipefail

# make sure we have access to loopback since we have many ~unit
# tests using the loopback interface
ip link set lo up

# Never switch toolchain. The caller resolved the exact go matching GOVERSION and
# passed it to us; we have no network in here, so a toolchain switch would fail.
# This also keeps the go driver and its tools on the same version, which matters
# since go1.25 builds tools such as covdata on demand instead of shipping them.
export GOTOOLCHAIN=local

# make sure we run all the "unit" tests (where "unit" means proper unit
# tests or tests using localhost or tests using netemx).
$1 test -short -race -count 1 -coverprofile=probe-cli.cov ./...
