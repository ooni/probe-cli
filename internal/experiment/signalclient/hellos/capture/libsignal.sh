#!/bin/bash
# Capture libsignal's ClientHellos by building libsignal's own chat_smoke_test example at the tags
# Signal's apps pin and running it against production (unauthenticated chat connections, then
# disconnect), rootless, with pasta writing the pcaps:
#   direct     ../libsignal.bin            the direct route to grpc.chat.signal.org, at LIBSIGNAL_TAG
#                                          (Signal-iOS's pin; Signal-Android's too when they agree);
#   reflector  ../libsignal-reflector.bin  the connection to a front on the reflector routes (proxyf,
#                                          proxyg), at LIBSIGNAL_TAG, captured on every front and
#                                          checked to be the same hello on each;
#   desktop    ../libsignal-desktop.bin    the direct route at LIBSIGNAL_DESKTOP_TAG, Signal-Desktop's
#                                          pin, when it differs from LIBSIGNAL_TAG; kept only when its
#                                          hello differs from libsignal.bin's (then hellos.go needs a
#                                          Desktop libsignal hello). Desktop never enables libsignal's
#                                          censorship circumvention, so it has no reflector hello.
# Usage: ./libsignal.sh [direct] [reflector] [desktop]   (default: all)
# Each tag's build takes several minutes and a few GB (BoringSSL, Rust); caches persist in $WORK.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd); HELLOS=$(dirname "$HERE")
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}; mkdir -p "$WORK/libsignal/cargo"
CTR=${CONTAINER:-podman}
: "${LIBSIGNAL_TAG:?run: eval \"\$(./versions.sh)\"}"

build() {  # tag: clone and build chat_smoke_test at tag (network allowed: cargo fetches crates)
  local src="$WORK/libsignal/src-$1"
  [ -d "$src" ] || git -c advice.detachedHead=false clone -q --depth 1 --branch "$1" https://github.com/signalapp/libsignal.git "$src"
  # bounded memory, cached registry and target dir
  "$CTR" run --rm --memory "${BUILD_MEMORY:-8g}" -v "$src:/src:Z" -v "$WORK/libsignal/cargo:/usr/local/cargo/registry:Z" \
    -w /src docker.io/library/rust:1-bookworm bash -c '
      set -e
      apt-get update -qq && apt-get install -y -qq cmake clang libclang-dev protobuf-compiler >/dev/null
      cargo build -q -p libsignal-net --example chat_smoke_test --features test-util'
}

smoke() {  # tag pcap args...: run chat_smoke_test at tag once against production, captured
  local src="$WORK/libsignal/src-$1" pcap=$2; shift 2; rm -f "$pcap"
  "$CTR" run --rm --network "pasta:--pcap,$pcap" -v "$src:/src:ro,Z" docker.io/library/rust:1-bookworm \
    /src/target/debug/examples/chat_smoke_test production "$@" 2>&1 | tail -2 || true
  chmod 644 "$pcap"
}

direct() {  # tag out.bin
  build "$1"
  smoke "$1" "$WORK/libsignal/direct-$1.pcap" --limit-to-routes direct
  python3 -I "$HERE/extract_hello.py" "$WORK/libsignal/direct-$1.pcap" grpc.chat.signal.org "$2"
}

# The reflector routes: --try-all-routes connects through each front of the route in turn. The fronts
# are libsignal's (rust/net/src/env.rs, PROXY_CONFIG_F_PROD and PROXY_CONFIG_G); the hello to a front
# does not depend on whether the tunnel then works.
reflector() {  # tag out.bin
  local fronts_proxyf="github.githubassets.com pinterest.com www.redditstatic.com"
  local fronts_proxyg="www.google.com android.clients.google.com clients3.google.com clients4.google.com googlemail.com"
  local bins=() route fronts sni pcap
  build "$1"
  for route in proxyf proxyg; do
    pcap="$WORK/libsignal/reflector-$1-$route.pcap"
    smoke "$1" "$pcap" --limit-to-routes "$route" --try-all-routes
    eval "fronts=\$fronts_$route"
    for sni in $fronts; do
      python3 -I "$HERE/extract_hello.py" "$pcap" "$sni" "$WORK/libsignal/reflector-$1-$sni.bin"
      bins+=("$WORK/libsignal/reflector-$1-$sni.bin")
    done
  done
  python3 -I "$HERE/same_hello.py" "${bins[@]}"
  cp "${bins[0]}" "$2"
}

for what in ${@:-direct reflector desktop}; do
  case $what in
  direct) direct "$LIBSIGNAL_TAG" "$HELLOS/libsignal.bin" ;;
  reflector) reflector "$LIBSIGNAL_TAG" "$HELLOS/libsignal-reflector.bin" ;;
  desktop)
    : "${LIBSIGNAL_DESKTOP_TAG:?run: eval \"\$(./versions.sh)\"}"
    if [ "$LIBSIGNAL_DESKTOP_TAG" = "$LIBSIGNAL_TAG" ]; then
      echo "libsignal.sh: Signal-Desktop pins $LIBSIGNAL_TAG too; libsignal.bin is its hello"; rm -f "$HELLOS/libsignal-desktop.bin"
    else
      direct "$LIBSIGNAL_DESKTOP_TAG" "$HELLOS/libsignal-desktop.bin"
      if cmp -s "$HELLOS/libsignal.bin" "$HELLOS/libsignal-desktop.bin"; then
        echo "libsignal.sh: $LIBSIGNAL_DESKTOP_TAG sends the same hello as $LIBSIGNAL_TAG; libsignal.bin is Desktop's too"
        rm "$HELLOS/libsignal-desktop.bin"
      else
        echo "libsignal.sh: $LIBSIGNAL_DESKTOP_TAG sends a different hello from $LIBSIGNAL_TAG: kept as libsignal-desktop.bin" >&2
      fi
    fi ;;
  *) echo "libsignal.sh: unknown capture $what" >&2; exit 2 ;;
  esac
done
echo "Captured with libsignal $LIBSIGNAL_TAG (Desktop: ${LIBSIGNAL_DESKTOP_TAG:-unset}): update the libsignal Hello names in hellos.go and the table in hellos/README.md"
