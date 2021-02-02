#!/bin/sh
set -e

# We don't have a git repository when running in github actions
v=`git describe --tags || echo $GITHUB_SHA`

case $1 in
  windows)
    set -x
    $0 __windows_amd64
    $0 __windows_386
    ;;

  __build_windows_amd64)
    # Note! This assumes we've installed the mingw-w64 compiler.
    set -x
    cd v3 && GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
      go build -ldflags='-s -w' -o ../ooniprobe.exe ./cmd/ooniprobe
    ;;

  __windows_amd64)
    set -x
    $0 __build_windows_amd64
    tar -cvzf ooniprobe_${v}_windows_amd64.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_amd64.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/amd64/
    ;;

  __build_windows_386)
    # Note! This assumes we've installed the mingw-w64 compiler.
    set -x
    cd v3 && GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc \
      go build -ldflags='-s -w' -o ../ooniprobe.exe ./cmd/ooniprobe
    ;;

  __windows_386)
    set -x
    $0 __build_windows_386
    tar -cvzf ooniprobe_${v}_windows_386.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_386.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/386/
    ;;

  linux)
    set -x
    $0 __linux_amd64
    $0 __linux_386
    ;;

  __linux_amd64)
    docker pull --platform linux/amd64 golang:1.14-alpine
    docker run --platform linux/amd64 -v`pwd`:/ooni -w/ooni golang:1.14-alpine \
      ./build.sh __alpine
    tar -cvzf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/amd64/
    ;;

  __linux_386)
    docker pull --platform linux/386 golang:1.14-alpine
    docker run --platform linux/386 -v`pwd`:/ooni -w/ooni golang:1.14-alpine \
      ./build.sh __alpine
    tar -cvzf ooniprobe_${v}_linux_386.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/386/
    ;;

  __alpine)
    set -x
    apk update
    apk upgrade
    apk add --no-progress gcc git linux-headers musl-dev
    $0 __build_linux_static
    ;;

  __build_linux_static)
    set -x
    cd v3 && go build -tags netgo -ldflags='-s -w -extldflags "-static"' \
      -o ../ooniprobe ./cmd/ooniprobe
    ;;

  __build_unix)
    # Note! The following lines _assumes_ you have a working C compiler. If you
    # have Xcode command line tools installed, you are fine.
    set -x
    cd v3 && go build -ldflags='-s -w' -o ../ooniprobe ./cmd/ooniprobe
    ;;

  macos|darwin)
    set -x
    $0 __build_unix
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
