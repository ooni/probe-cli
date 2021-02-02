#!/bin/sh
set -ex
go get -u github.com/shuLhan/go-bindata/...
gobindata=`go env GOPATH`/bin/go-bindata
version=`$gobindata -version | grep ^go-bindata | cut -d ' ' -f2`
if [ "$version" != "4.0.0" ]; then
  echo "FATAL: unexpected go-bindata version" 1>&2
  exit 1
fi
$gobindata -nometadata -o v3/cmd/ooniprobe/internal/bindata/bindata.go -pkg bindata data/...
