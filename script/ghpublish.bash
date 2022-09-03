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

# 3. determine whether this is a pre-release
prerelease="-p"
if [[ $__tag =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
	prerelease=""
fi

gh=${gh:-gh}

set -x

# 4. create the release as a pre-release unless it already exists
$gh release create $prerelease $__tag --target $GITHUB_SHA || true

# 5. publish all the assets passed as arguments to the target release
$gh release upload $__tag --clobber "$@"
