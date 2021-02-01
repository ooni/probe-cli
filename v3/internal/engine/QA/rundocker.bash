#!/bin/sh
set -ex
DOCKER=${DOCKER:-docker}
$DOCKER build -t jafar-qa ./QA/
$DOCKER run --privileged -v`pwd`:/jafar -w/jafar jafar-qa ./QA/pyrun.sh "$@"
