#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))
cd $reporoot

source $reporoot/MONOREPO/tools/libcore.bash

./MOBILE/android/newkeystore

(
	run export ANDROID_HOME=$(./MOBILE/android/home)
	run cd ./MONOREPO/repo/probe-android

	run ./gradlew assembleStableFullRelease

	apkdir=./app/build/outputs/apk/stableFull/release
	run cp -v $apkdir/app-stable-full-release-unsigned.apk  $reporoot/MOBILE/android/app-unsigned.apk
)

./MOBILE/android/sign
