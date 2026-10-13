#!/bin/bash
# Build and run one of signal_client's research tools from this checkout:
#   gotool.sh hellotool ARGS...    (hellos/capture/cmd/hellotool)
#   gotool.sh servertool ARGS...   (signalserver/cmd/servertool)
#   gotool.sh --path NAME          build it, and print the static binary's path (to mount in a container)
# Uses the host's Go if there is one, else a golang container. Binaries go to $GOTOOL_BIN.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
REPO=$(cd "$HERE/../../.." && pwd)
CTR=${CONTAINER:-podman}
BIN=${GOTOOL_BIN:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-tools}

print_path=false
if [ "${1:-}" = --path ]; then print_path=true; shift; fi
case "${1:-}" in
  hellotool) pkg=./internal/experiment/signalclient/hellos/capture/cmd/hellotool ;;
  servertool) pkg=./internal/experiment/signalclient/signalserver/cmd/servertool ;;
  *) echo "usage: gotool.sh [--path] hellotool|servertool ARGS..." >&2; exit 2 ;;
esac
name=$1; shift
mkdir -p "$BIN"

if command -v go >/dev/null; then
  (cd "$REPO" && CGO_ENABLED=0 go build -o "$BIN/$name" "$pkg")
else  # the Go version go.mod asks for; module and build caches persist in volumes
  "$CTR" run --rm -v "$REPO:/src:ro,Z" -v "$BIN:/out:Z" -v ooni-signal-gomod:/go/pkg/mod \
    -v ooni-signal-gobuild:/root/.cache/go-build -w /src -e CGO_ENABLED=0 \
    "docker.io/library/golang:$(sed -n 's/^go \([0-9]*\.[0-9]*\).*/\1/p' "$REPO/go.mod")" \
    go build -o "/out/$name" "$pkg"
fi

if $print_path; then echo "$BIN/$name"; else exec "$BIN/$name" "$@"; fi
