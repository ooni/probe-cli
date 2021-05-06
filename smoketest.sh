#!/bin/sh
set -ex
if [ "$#" != 1 ]; then
    echo "Usage: $0 <binary>" 1>&2
    exit 1
fi
$1 onboard --yes
# Important! DO NOT run performance from CI b/c it will overload m-lab servers
$1 run websites --config cmd/ooniprobe/testdata/testing-config.json --no-collector
