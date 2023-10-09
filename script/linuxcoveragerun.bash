#!/bin/bash

#
# Script invoked by ./script/linuxcoverage.bash to run tests
# with coverage using a separate network namespace with only loopback support.
#
# The first an unique argument is the path to the go binary to use.
#

set -euxo pipefail

# make sure we have access to loopback since we have many ~unit
# tests using the loopback interface
ip link set lo up

# make sure we run all the "unit" tests (where "unit" means proper unit
# tests or tests using localhost or tests using netemx).
$1 test -short -race -count 1 -coverprofile=probe-cli.cov ./...
