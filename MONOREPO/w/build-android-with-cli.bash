#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))
cd $reporoot

source $reporoot/MONOREPO/tools/libcore.bash

./MOBILE/android/newkeystore

make ./MOBILE/android
run cp -v MOBILE/android/oonimkall.aar ./MONOREPO/repo/probe-android/engine-experimental/

(
	run export ANDROID_HOME=$(./MOBILE/android/home)
	run cd ./MONOREPO/repo/probe-android
	# Note: we're building the experimental full release because the dev
	# release allows low-level code to do too many things. See
	# https://ooni.org/post/making-ooni-probe-android-more-resilient/#changing-our-android-tls-fingerprint
	run ./gradlew assembleExperimentalFullRelease
	apkdir=./app/build/outputs/apk/experimentalFull/release
	run cp -v $apkdir/app-experimental-full-release-unsigned.apk  $reporoot/MOBILE/android/app-unsigned.apk
)

./MOBILE/android/sign
