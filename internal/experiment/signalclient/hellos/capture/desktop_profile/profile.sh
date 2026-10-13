#!/bin/bash
# Profile Signal-Desktop on an unlinked install left on its QR screen (run fetch.sh first). The
# app is launched twice on the same profile: a first launch without capture (WARMUP seconds; it
# downloads the spellcheck dictionary and creates its database, which only a first launch does),
# then a second launch captured by pasta for DURATION seconds (default 240: past Signal-Server
# closing the provisioning socket after 90 s). FIRST=1 captures the first launch instead.
# Rootless podman, Xvfb; the app connects to Signal's production servers, as a new install does.
# Writes $WORK/desktop-profile/runs/<time>/: capture.pcap, keys.log (Chromium's TLS keys),
# the app's logs (home/.config/Signal/logs) and screenshots (they show a QR code; do not share).
# Then: ../../../gotool.sh hellotool analyze [-records] runs/<time>/capture.pcap
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}
B=$WORK/desktop-profile
WARMUP=${WARMUP:-30}
DURATION=${DURATION:-240}
RUN=$B/runs/$(date -u +%Y%m%dT%H%M%SZ)
mkdir -p "$RUN/home"
[ -e "$B/dl/signal-desktop.deb" ] || { echo "profile.sh: run fetch.sh first" >&2; exit 1; }
podman build -q -t signal-desktop-profile -f "$HERE/Containerfile" "$B" >/dev/null

# launch SECONDS NETWORK: run the app for SECONDS; screenshots at 20 and 60 s when it runs that long
launch() {
  podman run --rm --memory=3g --shm-size=1g --network "$2" \
    -v "$RUN:/out:Z" -e HOME=/out/home -e SSLKEYLOGFILE=/out/keys.log -e SECONDS_TO_RUN="$1" \
    signal-desktop-profile bash -c '
      Xvfb :99 -screen 0 1280x800x24 >/dev/null 2>&1 &
      sleep 1
      export DISPLAY=:99
      /opt/Signal/signal-desktop --no-sandbox >> /out/stdout.log 2>&1 &
      app=$!
      elapsed=0
      for t in 20 60; do
        [ "$SECONDS_TO_RUN" -ge "$t" ] || break
        sleep $((t - elapsed)); elapsed=$t
        import -window root /out/screen-${t}s.png 2>/dev/null || true
      done
      sleep $((SECONDS_TO_RUN - elapsed))
      kill $app 2>/dev/null; sleep 3'
}

if [ "${FIRST:-0}" = 1 ]; then
  echo "first launch (captured), ${DURATION}s"
else
  echo "first launch (not captured), ${WARMUP}s"
  launch "$WARMUP" pasta
  mkdir -p "$RUN/first"; mv "$RUN"/screen-*.png "$RUN/first/" 2>/dev/null || true
  mv "$RUN/keys.log" "$RUN/first/" 2>/dev/null || true
  echo "second launch (captured), ${DURATION}s"
fi
launch "$DURATION" "pasta:--pcap,$RUN/capture.pcap"
chmod -R u+rwX "$RUN"
chmod 644 "$RUN/capture.pcap"
echo "run: $RUN"
