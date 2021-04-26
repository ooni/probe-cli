#!/bin/bash
set -e
topdir=$(cd $(dirname $0) && pwd -P)
set -x
export PATH=$(go env GOPATH)/bin:$PATH
go get -u golang.org/x/mobile/cmd/gomobile
gomobile init
output=MOBILE/ios/oonimkall.framework
gomobile bind -target=ios -o $output -ldflags="-s -w" ./pkg/oonimkall
release=$(git describe --tags || echo $GITHUB_SHA)
version=$(date -u +%Y.%m.%d-%H%M%S)
podspecfile=./MOBILE/ios/oonimkall.podspec
(cd ./MOBILE/ios && rm -f oonimkall.framework.zip && zip -yr oonimkall.framework.zip oonimkall.framework)
podspectemplate=./MOBILE/template.podspec
cat $podspectemplate|sed -e "s/@VERSION@/$version/g" -e "s/@RELEASE@/$release/g" > $podspecfile
