#!/bin/sh

set -ex

./dist/${TRAVIS_OS_NAME}/amd64/ooniprobe onboard --yes
./dist/${TRAVIS_OS_NAME}/amd64/ooniprobe run --config testdata/testing-config.json -v --no-collector
