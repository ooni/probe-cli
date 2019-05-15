#!/bin/sh

set -ex

./dist/${TRAVIS_OS_NAME}/amd64/ooni onboard --yes
./dist/${TRAVIS_OS_NAME}/amd64/ooni run --no-collector
