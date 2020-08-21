#!/bin/sh
set -ex

case $1 in
  windows)
    v=`git describe --tags`
    # Note! This assumes we've installed the mingw-w64 compiler.
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
      go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_windows_amd64.tar.gz LICENSE.md Readme.md ooniprobe.exe
    zip ooniprobe_${v}_windows_amd64.zip LICENSE.md Readme.md ooniprobe.exe
    mv ooniprobe.exe ./CLI/windows/amd64/
    ;;

  linux)
    v=`git describe --tags`
    docker run -v`pwd`:/ooni -w/ooni golang:1.14-alpine ./build.sh _alpine
    tar -cvzf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/amd64/
    ;;

  _alpine)
    apk add --no-progress gcc git linux-headers musl-dev
    go build -tags netgo -ldflags='-s -w -extldflags "-static"' ./cmd/ooniprobe
    ;;

  macos)
    v=`git describe --tags`
    # Note! The following line _assumes_ you have a working C compiler. If you
    # have Xcode command line tools installed, you are fine.
    go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_macos_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/macos/amd64/
    ;;

  release)
    $0 linux
    $0 windows
    $0 macos
    ;;

  *)
    echo "Usage: $0 linux|macos|windows|release"
    ;;
esac
