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
	__tag=${__ref#refs/tags/}
else
	__tag=rolling
fi

set -x

# 3. create the release as a pre-release unless it already exists
gh release create -p $__tag --target $GITHUB_SHA || true

# 4. publish all the assets passed as arguments to the target release
gh release upload $__tag --clobber "$@"
