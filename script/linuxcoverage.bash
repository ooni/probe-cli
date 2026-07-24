#!/bin/bash

#
# Computes coverage inside an environment where we unshared the network namespace
# to ensure unit tests don't depend on the network.
#

set -euxo pipefail

# Use the pinned toolchain (see GOVERSION) to vendor.
./script/go.bash mod vendor

# Resolve the absolute path of the pinned go binary.
go=$HOME/sdk/go$(cat GOVERSION)/bin/go
if [[ ! -x $go ]]; then
	echo "FATAL: expected the pinned go toolchain at $go" 1>&2
	exit 1
fi

# run tests using a different network namespace
sudo unshare --net ./script/linuxcoveragerun.bash "$go"
