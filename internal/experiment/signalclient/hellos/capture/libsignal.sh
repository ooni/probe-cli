#!/bin/bash
# Capture libsignal's direct-route ClientHello by building libsignal's own chat_smoke_test example
# at the tag Signal's apps pin and running it once against production (an unauthenticated chat
# connection, then disconnect), rootless, with pasta writing the pcap. Writes ../libsignal.bin.
# The build takes several minutes and a few GB (BoringSSL, Rust); caches persist in $WORK.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd); HELLOS=$(dirname "$HERE")
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}; mkdir -p "$WORK/libsignal/cargo"
CTR=${CONTAINER:-podman}
: "${LIBSIGNAL_TAG:?run: eval \"\$(./versions.sh)\"}"
SRC="$WORK/libsignal/src-$LIBSIGNAL_TAG"
[ -d "$SRC" ] || git -c advice.detachedHead=false clone -q --depth 1 --branch "$LIBSIGNAL_TAG" https://github.com/signalapp/libsignal.git "$SRC"

# build (network allowed: cargo fetches crates); bounded memory, cached registry and target dir
"$CTR" run --rm --memory "${BUILD_MEMORY:-8g}" -v "$SRC:/src:Z" -v "$WORK/libsignal/cargo:/usr/local/cargo/registry:Z" \
  -w /src docker.io/library/rust:1-bookworm bash -c '
    set -e
    apt-get update -qq && apt-get install -y -qq cmake clang libclang-dev protobuf-compiler >/dev/null
    cargo build -q -p libsignal-net --example chat_smoke_test --features test-util'

# run once, captured
pcap="$WORK/libsignal/libsignal.pcap"; rm -f "$pcap"
"$CTR" run --rm --network "pasta:--pcap,$pcap" -v "$SRC:/src:ro,Z" docker.io/library/rust:1-bookworm \
  /src/target/debug/examples/chat_smoke_test production --limit-to-routes direct 2>&1 | tail -2
chmod 644 "$pcap"
python3 -I "$HERE/extract_hello.py" "$pcap" grpc.chat.signal.org "$HELLOS/libsignal.bin"
echo "Captured with libsignal $LIBSIGNAL_TAG: update HelloLibsignal's name in hellos.go and the table in hellos/README.md"
