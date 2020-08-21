#!/bin/sh
set -e
gobindata=`go env GOPATH`/bin/go-bindata
version=`$gobindata -version | grep go-bin | cut -d ' ' -f2`
if [ "$version" != "3.3.0" ]; then
    echo ""
    echo "FATAL: cannot find go-bindata binary in `go env GOPATH`"
    echo ""
    echo "Please install/update go-bindata with:"
    echo ""
    echo "  go get -u github.com/shuLhan/go-bindata/..."
    echo ""
    exit 1
fi
set -x
$gobindata -nometadata -o internal/bindata/bindata.go -pkg bindata data/...
