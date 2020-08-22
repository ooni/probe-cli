#!/bin/sh
set -ex
go get -u github.com/shuLhan/go-bindata/...
gobindata=`go env GOPATH`/bin/go-bindata
version=`$gobindata -version | grep go-bin | cut -d ' ' -f2`
if [ "$version" != "3.3.0" ]; then
  echo "FATAL: unexpected go-bindata version" 1>&2
  exit 1
fi
$gobindata -nometadata -o internal/bindata/bindata.go -pkg bindata data/...
