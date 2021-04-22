#!/bin/sh
set -e
# XXX: handle macos as darwin
case $1 in
  macos|darwin|linux|windows)
    set -x
    ./build miniooni --no-embed-psiphon --no-download-go "$1"
    ;;
  *)
    echo "usage: $0 darwin|linux|windows" 1>&2
    exit 1
esac
