#!/bin/sh
set -e

fatal() {
  echo ""
  echo "FATAL: cannot find go-bindata binary in `go env GOPATH`"
  echo ""
  echo "Please install/update go-bindata with:"
  echo ""
  echo "  go get -u github.com/shuLhan/go-bindata/..."
  echo ""
  exit 1
}

gobindata=`go env GOPATH`/bin/go-bindata
if [ ! -x $gobindata ]; then
  fatal
fi
version=`$gobindata -version | grep go-bin | cut -d ' ' -f2`
if [ "$version" != "3.3.0" ]; then
  fatal
fi
set -x
$gobindata -nometadata -o internal/bindata/bindata.go -pkg bindata data/...
