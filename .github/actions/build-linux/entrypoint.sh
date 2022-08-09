#!/bin/sh
# This script is executed by github actions when building inside
# an Alpine Linux docker container. Using Alpine Linux, which
# uses musl libc, allows us to emit static binaries.
set -euo pipefail

if [[ $DOCKERARCH == "arm64" || $DOCKERARCH == "arm64/v8" ]]; then
	GOARCH=arm64
	GOARM=
	archname=arm64
elif [[ $DOCKERARCH == arm/v7 ]]; then
	GOARCH=arm
	GOARM=7
	archname=armv7
elif [[ $DOCKERARCH == arm/v6 ]]; then
	GOARCH=arm
	GOARM=6
	archname=armv6
elif [[ $DOCKERARCH == 386 ]]; then
	GOARCH=386
	GOARM=
	archname=386
elif [[ $DOCKERARCH == amd64 ]]; then
	GOARCH=amd64
	GOARM=
	archname=amd64
else
	echo 'fatal: unknown architecture: $DOCKERARCH' 1>&2
	exit 1
fi

OONI_ENGINE=./internal/engine
OONI_CONFIG_KEY=$OONI_ENGINE/psiphon-config.key
OONI_CONFIG_JSON_AGE=$OONI_ENGINE/psiphon-config.json.age
if [[ -f $OONI_CONFIG_KEY && -f $OONI_CONFIG_JSON_AGE ]]; then
	OONI_PSIPHON_TAGS=ooni_psiphon_tags
else
	OONI_PSIPHON_TAGS=""
fi

package=$1
product=$(basename $package)

set -x

apk update
apk upgrade
apk add --no-progress gcc git linux-headers musl-dev

# some of the following exports are redundant but are however
# useful because they provide explicit logging
export GOARM=$GOARM
export CGO_ENABLED=1
export GOOS=linux
export GOARCH=$GOARCH

go build -o "./CLI/$product-linux-$archname" -tags=$OONI_PSIPHON_TAGS \
	-ldflags='-s -w -extldflags "-static"' $package
