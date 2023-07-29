#!/bin/bash

set -euxo pipefail

DOCKER=${DOCKER:-docker}

GOVERSION=$(cat GOVERSION)

cat > QA/Dockerfile << EOF
FROM golang:$GOVERSION-alpine
RUN apk add gcc go git musl-dev iptables tmux bind-tools curl sudo python3 util-linux valgrind
EOF

$DOCKER build -t jafar-qa ./QA/

$DOCKER run --privileged -v$(pwd):/jafar -w/jafar jafar-qa ./QA/dockermain.sh "$@"
