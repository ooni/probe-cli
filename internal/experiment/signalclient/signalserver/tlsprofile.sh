#!/bin/bash
# Measure the TLS/HTTP-2 profile of Signal's production front ends and of the local server started by
# run.sh, and print them side by side, so the local server can be kept matching production.
# Usage: tlsprofile.sh [prod|local|both]   (default both). Output: JSON in $WORK/tlsprofile-*.json
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
CTR=${CONTAINER:-podman}
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-server}
IMG=docker.io/library/python:3.12-slim   # OpenSSL 3.5: ML-KEM groups
run() {  # run NETWORK OUT TARGETS...
  local net=$1 out=$2; shift 2
  "$CTR" run --rm --network "$net" -v "$HERE:/h:ro,Z" -v signal-server-pip:/root/.cache/pip "$IMG" sh -c \
    'pip install -q --root-user-action=ignore h2 >/dev/null 2>&1; python3 -I /h/tlsprofile.py "$@"' sh "$@" > "$out"
  echo "wrote $out"
}
case "${1:-both}" in
  prod|both) run pasta "$WORK/tlsprofile-prod.json" chat.signal.org grpc.chat.signal.org ;;&
  local|both) run container:signal-server-local "$WORK/tlsprofile-local.json" \
                chat.signal.org@127.0.0.1:8443 grpc.chat.signal.org@127.0.0.1:8444 ;;
esac
