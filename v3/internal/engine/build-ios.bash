#!/bin/bash
set -e
topdir=$(cd $(dirname $0) && pwd -P)
set -x
export PATH=$(go env GOPATH)/bin:$PATH
export GO111MODULE=off
go get -u golang.org/x/mobile/cmd/gomobile
gomobile init
export GO111MODULE=on
output=MOBILE/ios/oonimkall.framework
gomobile bind -target=ios -o $output -ldflags="-s -w" ./oonimkall
