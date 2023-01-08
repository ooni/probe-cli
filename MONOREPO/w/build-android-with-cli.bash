#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))
cd $reporoot

source $reporoot/MONOREPO/tools/libcore.bash

./MOBILE/gomobile android ./pkg/oonimkall

run mv -v MOBILE/android/oonimkall.aar ./MONOREPO/repo/probe-android/engine-experimental/

(
	run export ANDROID_HOME=$(./MOBILE/android/home)
	run cd ./MONOREPO/repo/probe-android
	run ./gradlew assembleExperimentalFullRelease
	# output: ./app/build/outputs/apk/experimentalFull/release/app-experimental-full-release-unsigned.apk"
)

# TODO(bassosimone): we need to sign the APK with a suitable key.
