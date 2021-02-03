#!/bin/bash
set -e
cd $(dirname $0) 
case $1 in
  darwin)
    set -x
    go build -ldflags '-s -w' -buildmode=c-shared -o libooniffi.so .
    clang++ -std=c++11 -Wall -Wextra -I. -L. -o ffirun -looniffi ./testdata/ffirun.cpp
    ./ffirun testdata/webconnectivity.json
    ;;
  linux)
    set -x
    go build -ldflags '-s -w' -buildmode=c-shared -o libooniffi.so .
    g++ -std=c++11 -Wall -Wextra -I. -L. -o ffirun ./testdata/ffirun.cpp -looniffi
    LD_LIBRARY_PATH=. ./ffirun testdata/webconnectivity.json
    ;;
  windows)
    set -x
    go build -ldflags '-s -w' -buildmode=c-shared -o libooniffi.dll .
    x86_64-w64-mingw32-g++ -std=c++11 -Wall -Wextra -I. -L. -o ffirun.exe -looniffi ./testdata/ffirun.cpp
    ./ffirun.exe testdata/webconnectivity.json
    ;;
  *)
    echo "usage: $0 darwin|linux|windows" 1>&2
    exit 1
esac
