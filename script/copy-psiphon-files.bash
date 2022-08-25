#!/bin/bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
	echo "usage: $0 DIRECTORY" 1>&2
	exit 1
fi

GIT_CLONE_DIR=$1
mkdir -p $GIT_CLONE_DIR

OONIPRIVATE_DIR=$GIT_CLONE_DIR/github.com/ooni/probe-private
OONIPRIVATE_REPO=git@github.com:ooni/probe-private

if [[ ! -d $OONIPRIVATE_DIR ]]; then
	git clone $OONIPRIVATE_REPO $OONIPRIVATE_DIR
fi

cp $OONIPRIVATE_DIR/psiphon-config.key ./internal/engine
cp $OONIPRIVATE_DIR/psiphon-config.json.age ./internal/engine
