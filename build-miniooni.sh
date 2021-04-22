#!/bin/sh
set -e
case $1 in
  macos|darwin|linux|windows)
    set -x
    ./build miniooni --no-embed-psiphon --no-ooni-go "$1"
    ;;
  *)
    echo "usage: $0 darwin|linux|windows" 1>&2
    exit 1
esac
