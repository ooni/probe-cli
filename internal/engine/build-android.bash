#!/bin/bash
set -e
if [ -z "$ANDROID_HOME" -o "$1" = "--help" ]; then
    echo ""
    echo "usage: $0"
    echo ""
    echo "Please set ANDROID_HOME. We assume you have installed"
    echo "the Android SDK. You can do that on macOS using:"
    echo ""
    echo "    brew install --cask android-sdk"
    echo ""
    echo "Then make sure you install the required packages:"
    echo ""
    echo "sdkmanager --install 'build-tools;29.0.3' 'ndk;21.3.6528147'"
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
if [ -d $ANDROID_HOME/ndk-bundle ]; then
    echo ""
    echo "FATAL: currently we need 'ndk;21.3.6528147' instead of ndk-bundle"
    echo ""
    echo "See https://github.com/ooni/probe-engine/issues/1179."
    echo ""
    echo "To fix: sdkmanager --uninstall ndk-bundle"
    echo ""
    exit 1
fi
export ANDROID_NDK_HOME=$ANDROID_HOME/ndk/21.3.6528147
if [ ! -d $ANDROID_NDK_HOME ]; then
    echo ""
    echo "FATAL: currently we need 'ndk;21.3.6528147'"
    echo ""
    echo "See https://github.com/ooni/probe-engine/issues/1179."
    echo ""
    echo "To fix: sdkmanager --install 'ndk;21.3.6528147'"
    echo ""
    exit 1
fi

topdir=$(cd $(dirname $0) && pwd -P)
set -x
export PATH=$(go env GOPATH)/bin:$PATH
export GO111MODULE=off
go get -u golang.org/x/mobile/cmd/gomobile
gomobile init
export GO111MODULE=on
output=MOBILE/android/oonimkall.aar
gomobile bind -target=android -o $output -ldflags="-s -w" ./oonimkall
