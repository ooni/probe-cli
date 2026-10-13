#!/bin/bash
# Run the signalserver-tagged tests against the server started by run.sh. The tests share the
# server container's network namespace: 127.0.0.1:8443 is the chat.signal.org connector and
# 127.0.0.1:8444 the grpc.chat.signal.org one.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd); ROOT=$(cd "$HERE/../../../.." && pwd)
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-server}
podman run --rm --network container:signal-server-local -v "$ROOT:/src:Z" -v "$WORK/certs:/certs:ro,Z" \
  -v gomodcache:/go/pkg/mod -w /src -e SIGNAL_SERVER_ADDR=127.0.0.1:8443 \
  -e SIGNAL_SERVER_GRPC_ADDR=127.0.0.1:8444 -e SIGNAL_SERVER_CA=/certs/ca.pem \
  docker.io/library/golang:1.26 go test -tags signalserver -count=1 -v -run SignalServer ./internal/experiment/signalclient/
