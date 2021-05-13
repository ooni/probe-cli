#!/bin/sh
#
# This test for now uses --no-collector and we just ensure that the OONI
# instance is not exploding. We are confident that, if miniooni submits
# measurements, also ooniprobe should be able to do that. However, it would
# actually be nice if someone could enhance this script to also make sure
# that we can actually fetch the measurements we submit.
#
set -ex
if [ "$#" != 1 ]; then
    echo "Usage: $0 <binary>" 1>&2
    exit 1
fi
$1 onboard --yes
# Important! DO NOT run performance from CI b/c it will overload m-lab servers
$1 run websites --config cmd/ooniprobe/testdata/testing-config.json --no-collector
