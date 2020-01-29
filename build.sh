#!/bin/bash
set -e

buildtags=""
ldflags="-s -w"

if [ "$1" = "bindata" ]; then
    GO_BINDATA_V=$(go-bindata -version | grep go-bin | cut -d ' ' -f2)
    [[ "$GO_BINDATA_V" == "3.2.0" ]] && echo "Updating bindata" || exit "Wrong go-bindata-version"
    go-bindata -nometadata -o internal/bindata/bindata.go -pkg bindata data/...
    echo "DONE"
    exit 0
fi

if [ "$1" = "windows" ]; then
  set -x
  CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++                         \
    CGO_ENABLED=1 GOOS=windows GOARCH=amd64                                    \
      go build $buildtags -ldflags="$ldflags"                                  \
        -o dist/windows/amd64/ooniprobe.exe -v ./cmd/ooniprobe

elif [ "$1" = "linux" ]; then
  set -x
  $0 __docker go build $buildtags -ldflags="$ldflags"                          \
      -o dist/linux/amd64/ooniprobe -v ./cmd/ooniprobe

elif [ "$1" = "macos" ]; then
  set -x
  go build $buildtags -ldflags="$ldflags"                                      \
    -o dist/macos/amd64/ooniprobe -v ./cmd/ooniprobe

elif [ "$1" = "release" ]; then
  set -x
  v=`git describe --tags`
  $0 linux
  tar -czf ooniprobe_${v}_linux_amd64.tar.gz LICENSE.md Readme.md              \
    -C ./dist/linux/amd64 ooniprobe
  shasum -a 256 ooniprobe_${v}_linux_amd64.tar.gz > ooniprobe_checksums.txt
  $0 macos
  tar -czf ooniprobe_${v}_darwin_amd64.tar.gz LICENSE.md Readme.md             \
    -C ./dist/macos/amd64 ooniprobe
  shasum -a 256 ooniprobe_${v}_darwin_amd64.tar.gz >> ooniprobe_checksums.txt
  $0 windows
  tar -czf ooniprobe_${v}_windows_amd64.tar.gz ./dist/windows/amd64            \
    -C dist/windows/amd64 ooniprobe.exe
  shasum -a 256 ooniprobe_${v}_windows_amd64.tar.gz >> ooniprobe_checksums.txt
  echo ""
  echo "Now sign ooniprobe_checksums.txt and upload it along with tarballs to GitHub"

elif [ "$1" = "__docker" ]; then
  set -x
  shift
  docker build -t oonibuild .
  docker run -v `pwd`:/oonibuild                                               \
             -w /oonibuild                                                     \
             -t                                                                \
             --cap-drop=all                                                    \
             --user `id -u`:`id -g`                                            \
             -e 'GOCACHE=/oonibuild/testdata/gotmp/cache'                      \
             -e 'GOPATH=/oonibuild/testdata/gotmp/path'                        \
             -e "TRAVIS_JOB_ID=$TRAVIS_JOB_ID"                                 \
             -e "TRAVIS_PULL_REQUEST=$TRAVIS_PULL_REQUEST"                     \
             oonibuild "$@"

elif [ "$1" = "_travis-linux" ]; then
  set -x
  $0 linux
  # TODO -race does not work on alpine.
  # See: https://travis-ci.org/ooni/probe-cli/builds/619631256#L962
  $0 __docker go get -v golang.org/x/tools/cmd/cover
  $0 __docker go get -v github.com/mattn/goveralls
  $0 __docker go test -v -coverprofile=coverage.cov -coverpkg=./... ./...
  $0 __docker /oonibuild/testdata/gotmp/path/bin/goveralls                     \
          -coverprofile=coverage.cov -service=travis-ci

elif [ "$1" = "_travis-osx" ]; then
  set -x
  brew tap measurement-kit/measurement-kit
  brew update
  brew upgrade
  brew install measurement-kit
  $0 macos
  go test -v -race -coverprofile=coverage.cov -coverpkg=./... ./...

elif [ "$1" = "help" ]; then
  echo "Usage: $0 linux | macos | release | windows"
  echo ""
  echo "Builds OONI on supported systems. The output binary will"
  echo "be saved at './dist/<system>/<arch>/ooniprobe[.exe]'."
  echo ""
  echo "# Linux"
  echo ""
  echo "To compile for Linux we use a docker container with the binary"
  echo "Measurement Kit dependency installed. So you need docker installed."
  echo ""
  echo "# macOS"
  echo ""
  echo "You must be on macOS. You must install Measurement Kit once using:"
  echo ""
  echo "- brew tap measurement-kit/measurement-kit"
  echo "- brew install measurement-kit"
  echo ""
  echo "You should keep Measurement Kit up-to-date using:"
  echo ""
  echo "- brew upgrade"
  echo ""
  echo "# Release"
  echo ""
  echo "Will build ooniprobe for all supported systems."
  echo ""
  echo "# Windows"
  echo ""
  echo "You must be on macOS. You must install Measurement Kit once using:"
  echo ""
  echo "- brew tap measurement-kit/measurement-kit"
  echo "- brew install mingw-w64-measurement-kit"
  echo ""
  echo "You should keep Measurement Kit up-to-date using:"
  echo ""
  echo "- brew upgrade"
  echo ""

else
  echo "Invalid usage; try '$0 help' for more help." 1>&2
  exit 1
fi
