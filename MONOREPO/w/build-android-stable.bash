#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))
cd $reporoot

source $reporoot/MONOREPO/tools/libgit.bash
for_each_repo fail_if_dirty
for_each_repo fail_if_not_main

run ./MOBILE/android/newkeystore

(
	run export ANDROID_HOME=$(./MOBILE/android/home)
	run cd ./MONOREPO/repo/probe-android

	run ./gradlew assembleStableFullRelease

	apkdir=./app/build/outputs/apk/stableFull/release
	run cp -v $apkdir/app-stable-full-release-unsigned.apk  $reporoot/MOBILE/android/app-unsigned.apk
)

run ./MOBILE/android/sign
