#!/bin/bash
set -euo pipefail

reporoot=$(dirname $(dirname $(dirname $(realpath $0))))

source $reporoot/MONOREPO/tools/libcore.bash

FATALMSG="this script only runs in linux/amd64, darwin/amd64, darwin/arm64"
GOOS=$(goos)
GOARCH=$(goarch)

case $GOOS in
linux | darwin) ;;
*)
	fatal $FATALMSG
	;;
esac

case $GOARCH in
amd64) ;;
arm64)
	if [[ $GOOS != darwin ]]; then
		fatal $FATALMSG
	fi
	GOARCH=amd64 # there's no ooni/probe-desktop for arm64
	;;
*)
	fatal $FATALMSG
	;;
esac

run ./CLI/go-build-generic ./cmd/ooniprobe
run mkdir -p ./MONOREPO/repo/probe-desktop/build/probe-cli/${GOOS}_${GOARCH}
run mv -v ooniprobe ./MONOREPO/repo/probe-desktop/build/probe-cli/${GOOS}_${GOARCH}
(
	run cd ./MONOREPO/repo/probe-desktop
	run yarn install
	run yarn dev
)
