#!/bin/bash
# Capture Signal-Desktop's own-fetch ClientHello by running desktop-fetch.js under the Electron
# version Signal-Desktop pins (checked against the release's SHASUMS256.txt), rootless, with pasta
# writing the pcap. Writes ../desktop-electron.bin.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd); HELLOS=$(dirname "$HERE")
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}; mkdir -p "$WORK/desktop"
CTR=${CONTAINER:-podman}
HELLOTOOL=$("$HELLOS/../gotool.sh" --path hellotool)  # built from this checkout
: "${ELECTRON_VERSION:?run: eval \"\$(./versions.sh)\"}"
V=$ELECTRON_VERSION; Z="electron-v$V-linux-x64.zip"; D="$WORK/desktop/electron-$V"
if [ ! -x "$D/electron" ]; then
  base="https://github.com/electron/electron/releases/download/v$V"
  curl -fsSL -o "$WORK/desktop/$Z" "$base/$Z"
  curl -fsSL -o "$WORK/desktop/SHASUMS256-$V.txt" "$base/SHASUMS256.txt"
  (cd "$WORK/desktop" && grep " \*\?$Z\$" "SHASUMS256-$V.txt" | sed 's/ \*/  /' | sha256sum -c --quiet)
  python3 -I -c 'import sys, zipfile, os
z = zipfile.ZipFile(sys.argv[1]); z.extractall(sys.argv[2])
for n in ("electron", "chrome_crashpad_handler", "chrome-sandbox"): os.chmod(os.path.join(sys.argv[2], n), 0o755)' "$WORK/desktop/$Z" "$D"
  rm -f "$WORK/desktop/$Z"
fi
"$CTR" build -q -t localhost/signal-hello-electron -f "$HERE/electron.Containerfile" "$HERE" >/dev/null
pcap="$WORK/desktop/desktop.pcap"; rm -f "$pcap"
"$CTR" run --rm --network "pasta:--pcap,$pcap" -e ELECTRON_RUN_AS_NODE=1 -e SIGNAL_CA=/hellos/signal.cer \
  -v "$D:/electron:ro,Z" -v "$HERE:/src:ro,Z" -v "$HELLOS:/hellos:ro,Z" localhost/signal-hello-electron \
  /electron/electron /src/desktop-fetch.js https://updates2.signal.org/desktop/latest-linux.yml
chmod 644 "$pcap"
"$HELLOTOOL" extract "$pcap" updates2.signal.org "$HELLOS/desktop-electron.bin"
echo "Captured with Electron $V: update HelloDesktop's name in hellos.go and the table in hellos/README.md"
