#!/bin/bash
set -euxo pipefail
# If this script is invoked by ./script/go.bash, then go is
# the correct version of go expected by the buildtool.
#
# See https://github.com/ooni/probe/issues/2664
go "$@"
