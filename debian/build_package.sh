#!/bin/bash
set -eux
#
# Build ooniprobe-cli package
#
# Requires a probe binary at ./dist/linux/amd64/ooniprobe
#
test -f ./dist/linux/amd64/ooniprobe
sudo apt-get update -q
sudo apt-get build-dep -y --no-install-recommends .
dpkg-buildpackage -us -uc -b
