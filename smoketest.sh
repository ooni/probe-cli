#!/bin/sh
set -ex
./dist/$1/amd64/ooniprobe onboard --yes
./dist/$1/amd64/ooniprobe run --config testdata/testing-config.json -v --no-collector
