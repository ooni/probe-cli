#!/bin/bash
set -e
if [ -z "$ANDROID_HOME" -o "$1" = "--help" ]; then
    echo ""
    echo "usage: $0 [--sign]"
    echo ""
    echo "Please set ANDROID_HOME. We assume you have installed"
    echo "the Android SDK. You can do that on macOS using:"
    echo ""
    echo "    brew install --cask android-sdk"
    echo ""
    echo "Then make sure you install the required packages:"
    echo ""
    echo "sdkmanager --install 'build-tools;29.0.3' 'ndk-bundle'"
    echo ""
    echo "or, if you already installed, that you're up to date:"
    echo ""
    echo "sdkmanager --update"
    echo ""
    echo "Once you have done that, please export ANDROID_HOME to"
    echo "point to /usr/local/Caskroom/android-sdk/<version>."
    echo ""
    exit 1
fi
topdir=$(cd $(dirname $0) && pwd -P)
set -x
export PATH=$(go env GOPATH)/bin:$PATH
go get -u golang.org/x/mobile/cmd/gomobile
gomobile init
version=$(date -u +%Y.%m.%d-%H%M%S)
aarfile=MOBILE/android/oonimkall-$version.aar
gomobile bind -target=android -o $aarfile -ldflags="-s -w" ./pkg/oonimkall
pomfile=./MOBILE/android/oonimkall-$version.pom
pomtemplate=./MOBILE/template.pom
cat $pomtemplate|sed "s/@VERSION@/$version/g" > $pomfile
if [ "$1" == "--sign" ]; then
  cd ./MOBILE/android
  for file in `ls *`; do
    gpg -ab -u simone@openobservatory.org $file
  done
  jar -cvf bundle.jar oonimkall-$version*
fi
