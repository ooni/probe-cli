#!/bin/bash
# Measure the TLS/HTTP-2 profile of Signal's production front ends and of the local server started by
# run.sh, with `servertool tlsprofile` run where OpenSSL 3.5 is (for ML-KEM groups).
# Usage: tlsprofile.sh [prod|local|both]   (default both). Writes $WORK/tlsprofile-{prod,local}.json
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
CTR=${CONTAINER:-podman}
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-server}; mkdir -p "$WORK"
SERVERTOOL=$("$HERE/../gotool.sh" --path servertool)
run() {  # run NETWORK OUT TARGETS...
  local net=$1 out=$2; shift 2
  "$CTR" run --rm --network "$net" -v "$SERVERTOOL:/servertool:ro,Z" docker.io/library/python:3.12-slim \
    /servertool tlsprofile "$@" > "$out.tmp"
  mv "$out.tmp" "$out"
  echo "wrote $out"
}
case "${1:-both}" in
  prod|local|both) ;;
  *) echo "usage: tlsprofile.sh [prod|local|both]" >&2; exit 2 ;;
esac
case "${1:-both}" in
  prod|both) run pasta "$WORK/tlsprofile-prod.json" chat.signal.org grpc.chat.signal.org ;;&
  local|both) run container:signal-server-local "$WORK/tlsprofile-local.json" \
                chat.signal.org@127.0.0.1:8443 grpc.chat.signal.org@127.0.0.1:8444 ;;
esac
