#!/bin/bash
set -euxo pipefail

# Make sure we're not going to execute gh
export gh="echo gh"

# Use a very old SHA as target
export GITHUB_SHA="7327e1ff7f0cfdc5ff0335574b85dc8ceb9465b6"

# Test 1: make sure we're publishing to rolling as a
# pre-release when the build targets a branch
export GITHUB_REF="refs/heads/feature"
./script/ghpublish.bash ABC > ghpublish.out.txt
diff ./script/ghpublish-branch.out.txt ghpublish.out.txt

# Test 2: make sure we're publishing to rolling as a
# pre-release when the build target is a PR
export GITHUB_REF="refs/pull/123/merge"
./script/ghpublish.bash ABC > ghpublish.out.txt
diff ./script/ghpublish-pr.out.txt ghpublish.out.txt

# Test 3: make sure we're publishing to a pre-release when
# we're building a tag that is not a stable release.
export GITHUB_REF="refs/tags/v0.0.0-alpha"
./script/ghpublish.bash ABC > ghpublish.out.txt
diff ./script/ghpublish-prerelease.out.txt ghpublish.out.txt

# Test 3: make sure we're publishing to a release when
# we're building a tag that is a stable release.
export GITHUB_REF="refs/tags/v0.0.0"
./script/ghpublish.bash ABC > ghpublish.out.txt
diff ./script/ghpublish-release.out.txt ghpublish.out.txt
