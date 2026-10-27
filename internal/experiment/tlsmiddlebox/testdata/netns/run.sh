#!/bin/bash
# Runs TestNetns (../../netns_test.go) without root, in rootless podman (or ENGINE=docker):
#
#   internal/experiment/tlsmiddlebox/testdata/netns/run.sh [-test.run 'TestNetns/no_middlebox']
#
# Builds the test binary from this checkout in a golang container (GOVERSION), then runs it in
# a privileged container where topo.sh builds the namespaces. Extra arguments go to the test
# binary. GOMOD_VOLUME and GOBUILD_VOLUME name the volumes that cache Go modules and builds.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
ROOT=$(cd "$HERE/../../../../.." && pwd)
ENGINE=${ENGINE:-podman}
OUT=$(mktemp -d)
trap 'rm -rf "$OUT"' EXIT

"$ENGINE" run --rm -v "$ROOT:/src:ro,Z" -v "$OUT:/out:Z" \
  -v "${GOMOD_VOLUME:-tlsmiddlebox-netns-gomod}:/go/pkg/mod" \
  -v "${GOBUILD_VOLUME:-tlsmiddlebox-netns-gobuild}:/root/.cache/go-build" \
  -w /src -e CGO_ENABLED=0 "docker.io/library/golang:$(cat "$ROOT/GOVERSION")" \
  go test -c -tags netns -o /out/tlsmiddlebox.test ./internal/experiment/tlsmiddlebox
"$ENGINE" build -q -t tlsmiddlebox-netns "$HERE" >/dev/null
"$ENGINE" run --rm --privileged -v "$HERE:/netns:ro,Z" -v "$OUT:/out:ro,Z" tlsmiddlebox-netns \
  /netns/topo.sh /out/tlsmiddlebox.test -test.run '^TestNetns$' -test.v "$@"
