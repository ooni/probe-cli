#!/bin/sh
set -e

# We don't have a git repository when running in github actions
v=`git describe --tags || echo $GITHUB_SHA`

case $1 in
  windows)
    set -x
    $0 windows_amd64
    $0 windows_386
    ;;

  windows_amd64)
    # Note! This assumes we've installed the mingw-w64 compiler.
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
      go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_windows_amd64.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_amd64.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/amd64/
    ;;

  windows_386)
    # Note! This assumes we've installed the mingw-w64 compiler.
    GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc \
      go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_windows_386.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_386.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/386/
    ;;

  linux)
    set -x
    $0 linux_amd64
    $0 linux_386
    ;;

  linux_amd64)
    docker pull --platform linux/amd64 golang:1.14-alpine
    docker run --platform linux/amd64 -v`pwd`:/ooni -w/ooni golang:1.14-alpine ./build.sh _alpine
    tar -cvzf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/amd64/
    ;;

  linux_386)
    docker pull --platform linux/386 golang:1.14-alpine
    docker run --platform linux/386 -v`pwd`:/ooni -w/ooni golang:1.14-alpine ./build.sh _alpine
    tar -cvzf ooniprobe_${v}_linux_386.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/386/
    ;;

  _alpine)
    apk update
    apk upgrade
    apk add --no-progress gcc git linux-headers musl-dev
    go build -tags netgo -ldflags='-s -w -extldflags "-static"' ./cmd/ooniprobe
    ;;

  macos|darwin)
    set -x
    # Note! The following line _assumes_ you have a working C compiler. If you
    # have Xcode command line tools installed, you are fine.
    go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_darwin_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/darwin/amd64/
    ;;

  release)
    $0 linux
    $0 windows
    $0 darwin
    shasum -a 256 ooniprobe_${v}_*_*.* > ooniprobe_checksums.txt
    ;;

  *)

    set +x
    echo "Usage: $0 darwin|linux|macos|windows|release"
    echo ""
    echo "You need a C compiler and Go >= 1.14. The C compiler must be a"
    echo "UNIX like compiler like GCC, Clang, Mingw-w64."
    echo ""
    echo "To build a static Linux binary, we use Docker and Alpine. We currently"
    echo "build for linux/386 and linux/amd64."
    echo ""
    echo "You can cross compile for Windows from macOS or Linux. You can"
    echo "compile for Linux as long as you have Docker. Cross compiling for"
    echo "macOS has never been tested. We have a bunch of cross compiling"
    echo "checks inside the .github/workflows/cross.yml file."
    echo ""
    echo "The macos rule is an alias for the darwin rule. The generated"
    echo 'binary file is named ooniprobe_${version}_darwin_${arch}.tar.gz'
    echo "because the platform name is darwin."
    echo ""
    ;;
esac
