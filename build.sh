#!/bin/bash
set -e

# We don't have a git repository when running in github actions
if command -v git &> /dev/null; then
  v=$(git describe --tags || echo $GITHUB_SHA)
else
  v=GITHUB_SHA
fi

ci_build_on_debian() {
  # Install build dependencies, go, and build the probe
  set -eux
  case $ARCH in
    armv6) HDR=armmp;;
    armv7) HDR=armmp;;
    aarch64) HDR=arm64;;
    *) HDR=$ARCH;;
  esac
  uname -a
  apt-get -qq update
  apt-get install -qq -y --no-install-recommends gcc git linux-headers-$HDR wget musl-dev ca-certificates libc6-dev

  case $ARCH in
    armv6) GOARCH=armv6l;; #ARMv6
    armv7) GOARCH=armv6l;; #ARMv7: not available, use v6
    aarch64) GOARCH=arm64;; #ARMv8
    *) GOARCH=$ARCH;;
  esac
  tarfn=go1.16.linux-$GOARCH.tar.gz
  echo Downloading $tarfn
  wget --no-verbose https://golang.org/dl/$tarfn
  tar xfz $tarfn
  export PATH=$PATH:$(pwd)/go/bin
  go version
  go env

  go build -tags netgo -ldflags='-s -w -extldflags "-static"' ./cmd/ooniprobe
}

ci_build_upload_deb() {
  # Build and upload .deb package
  set -eu
  for e in ARCH BT_APIKEY GITHUB_REF GITHUB_RUN_NUMBER; do
    [[ -z "${!e}" ]] && echo "Please set env var $e" && exit 1
  done
  case $ARCH in
    armv6) DARCH=armel;;
    armv7) DARCH=armhf;;
    *) DARCH=$ARCH;;
  esac
  DEBDIST=unstable
  BT_APIUSER=federicoceratto
  BT_ORG=ooni
  BT_PKGNAME=ooniprobe
  apt-get -qq update
  # apt-get build-dep -y --no-install-recommends .
  apt-get install -q  -y --no-install-recommends dpkg-dev build-essential devscripts debhelper
  VER=$(./ooniprobe version)
  if [[ ! $GITHUB_REF =~ ^refs/tags/* ]]; then
    VER="${VER}~${GITHUB_RUN_NUMBER}"
    dch -v $VER "New test version"
    BT_REPO=ooniprobe-debian-test
  else
    dch -v $VER "New release"
    BT_REPO=ooniprobe-debian
  fi
  dpkg-buildpackage -us -uc -b
  find ../ -name "*.deb" -type f
  DEB="../ooniprobe-cli_${VER}_${DARCH}.deb"
  BT_FNAME="ooniprobe-cli_${VER}_${DARCH}.deb"
  curl --upload-file "${DEB}" -u "${BT_APIUSER}:${BT_APIKEY}" \
            "https://api.bintray.com/content/${BT_ORG}/${BT_REPO}/${BT_PKGNAME}/${VER}/${BT_FNAME};deb_distribution=${DEBDIST};deb_component=main;deb_architecture=${DARCH};publish=1"
}

case $1 in
  windows)
    set -x
    $0 windows_amd64
    $0 windows_386
    ;;

  windows_amd64)
    go run ./internal/cmd/getresources
    # Note! This assumes we've installed the mingw-w64 compiler.
    GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc \
      go build -ldflags='-s -w' ./cmd/ooniprobe
    tar -cvzf ooniprobe_${v}_windows_amd64.tar.gz LICENSE.md Readme.md ooniprobe.exe
    # We don't have zip inside the github actions runner
    zip ooniprobe_${v}_windows_amd64.zip LICENSE.md Readme.md ooniprobe.exe || true
    mv ooniprobe.exe ./CLI/windows/amd64/
    ;;

  windows_386)
    go run ./internal/cmd/getresources
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
    go run ./internal/cmd/getresources
    docker pull --platform linux/amd64 golang:1.16-alpine
    docker run --platform linux/amd64 -v`pwd`:/ooni -w/ooni golang:1.16-alpine ./build.sh _alpine
    tar -cvzf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md ooniprobe
    mv ooniprobe ./CLI/linux/amd64/
    ;;

  linux_386)
    go run ./internal/cmd/getresources
    docker pull --platform linux/386 golang:1.16-alpine
    docker run --platform linux/386 -v`pwd`:/ooni -w/ooni golang:1.16-alpine ./build.sh _alpine
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
    go run ./internal/cmd/getresources
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

  _ci_build_on_debian)
    ci_build_on_debian
    ;;

  _ci_build_upload_deb)
    ci_build_upload_deb
    ;;

  *)

    set +x
    echo "Usage: $0 darwin|linux|macos|windows|release"
    echo ""
    echo "You need a C compiler and Go >= 1.16. The C compiler must be a"
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
