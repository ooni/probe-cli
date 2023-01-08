#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))
cd $reporoot

source $reporoot/MONOREPO/tools/libcore.bash

./MOBILE/android/newkeystore

./MOBILE/gomobile android ./pkg/oonimkall

run cp -v MOBILE/android/oonimkall.aar ./MONOREPO/repo/probe-android/engine-experimental/

(
	run export ANDROID_HOME=$(./MOBILE/android/home)
	run cd ./MONOREPO/repo/probe-android
	run ./gradlew assembleExperimentalFullRelease
	apkdir=./app/build/outputs/apk/experimentalFull/release
	run cp -v $apkdir/app-experimental-full-release-unsigned.apk  $reporoot/MOBILE/android/app-unsigned.apk
)

./MOBILE/android/sign
