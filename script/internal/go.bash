#!/bin/bash
set -euxo pipefail
# If this script is invoked by ./script/go.bash, then go is
# the correct version of go expected by the buildtool.
go "$@"
