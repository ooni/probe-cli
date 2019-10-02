#!/bin/sh

set -ex

./dist/${OS_NAME}/amd64/ooniprobe onboard --yes
./dist/${OS_NAME}/amd64/ooniprobe run --config testdata/testing-config.json -v --no-collector
