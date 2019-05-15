#!/bin/sh

set -ex

./dist/${OS_NAME}/amd64/ooni onboard --yes
./dist/${OS_NAME}/amd64/ooni run -v --no-collector
