#!/bin/bash

#
# Script invoked by ./script/linuxcoverage.bash to run tests
# with coverage using a separate namespace with loopback
#
# The first an unique argument is the path to the go binary to use.
#

set -euxo pipefail

# make sure we have access to loopback since we have many ~unit
# tests using the loopback interface
ip link set lo up

$1 test -short -race -coverprofile=probe-cli.cov ./...
