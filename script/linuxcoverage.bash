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

# Since go1.25 the toolchain no longer ships covdata as a prebuilt binary. When
# `go test -cover` needs it for a package that has no test files, go tries to obtain
# it by downloading a toolchain, which cannot work inside the network namespace we
# enter below. See https://github.com/golang/go/issues/75031, which upstream only 
# fixes in go1.27.
tooldir=$($go env GOROOT)/pkg/tool/$($go env GOOS)_$($go env GOARCH)
if [[ ! -x $tooldir/covdata ]]; then
	$go build -o "$tooldir/covdata" cmd/covdata
fi

# run tests using a different network namespace
sudo unshare --net ./script/linuxcoveragerun.bash "$go"
