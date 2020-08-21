#!/bin/sh
set -ex

# We don't have a git repository when running in github actions
v=`git describe --tags || echo $GITHUB_SHA`

case $1 in
  windows)
    # Note! This assumes we've installed the mingw-w64 compiler.
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
      go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_windows_amd64.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_amd64.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/amd64/
    ;;

  linux)
    docker run -v`pwd`:/ooni -w/ooni golang:1.14-alpine ./build.sh _alpine
    tar -cvzf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/amd64/
    ;;

  _alpine)
    apk add --no-progress gcc git linux-headers musl-dev
    go build -tags netgo -ldflags='-s -w -extldflags "-static"' ./cmd/ooniprobe
    ;;

  macos)
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
    echo ""
    echo "You need a C compiler and Go >= 1.14. The C compiler must be a"
    echo "UNIX like compiler like GCC, Clang, Mingw-w64."
    echo ""
    echo "To build a static Linux binary, we use Docker and Alpine."
    echo ""
    echo "You can cross compile for Windows from macOS or Linux. You can"
    echo "compile for Linux as long as you have Docker. Cross compiling for"
    echo "macOS has never been tested."
    echo ""
    ;;
esac
