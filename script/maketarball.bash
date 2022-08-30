#!/bin/bash

set -euo pipefail

# 1. obtain the github ref of this action run
__ref=${GITHUB_REF:-}

if [[ $__ref == "" ]]; then
	echo "FATAL: missing github ref" 1>&2
	exit 1
fi

# 2. determine whether to publish to a release or to rolling
if [[ $__ref =~ ^refs/tags/v ]]; then
	__version=${__ref#refs/tags/v}
else
	__version=rolling
fi

set -x

# 3. make sure we're using the correct go version
./CLI/check-go-version

# 4. generate the actual tarball
go mod vendor
tar -czf ooni-probe-cli-${__version}.tar.gz --transform "s,^,ooni-probe-cli-${__version}/," *

