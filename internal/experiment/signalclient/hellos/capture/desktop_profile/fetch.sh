#!/bin/bash
# Download the current Signal-Desktop .deb from Signal's APT repository and verify it: InRelease's
# signature with Signal's key (keys.asc, in a throwaway GNUPGHOME), the Packages index's SHA-256
# against InRelease, and the .deb's SHA-256 against Packages. Prints the signing key's
# fingerprint (DBA36B5181D0C816F630E889D980A17457F6FB06 in 2026-10) and the version.
# Writes $WORK/desktop-profile/dl/signal-desktop.deb.
set -euo pipefail
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}
D=$WORK/desktop-profile/dl; mkdir -p "$D"
REPO=https://updates.signal.org/desktop/apt
cd "$D"
curl -fsSL -o keys.asc "$REPO/keys.asc"
curl -fsSL -o InRelease "$REPO/dists/xenial/InRelease"
export GNUPGHOME=$D/gnupg; mkdir -p "$GNUPGHOME"; chmod 700 "$GNUPGHOME"
gpg -q --import keys.asc
gpg --with-colons --fingerprint | awk -F: '/^fpr/{print "signing key fingerprint:", $10; exit}'
gpg -q --verify InRelease 2>&1 | grep -E "Good signature" || { echo "fetch.sh: InRelease has no good signature" >&2; exit 1; }
gpg -q --decrypt InRelease 2>/dev/null > Release

curl -fsSL -o Packages "$REPO/dists/xenial/main/binary-amd64/Packages"
want=$(awk '/^SHA256:/{f=1;next} f && $3=="main/binary-amd64/Packages"{print $1; exit}' Release)
echo "$want  Packages" | sha256sum -c --quiet && echo "Packages: SHA-256 matches Release"

# the newest signal-desktop entry (not signal-desktop-beta)
python3 -I - Packages > deb.txt <<'EOF'
import sys
entries = [dict(l.split(": ", 1) for l in block.splitlines() if ": " in l)
           for block in open(sys.argv[1]).read().split("\n\n") if block.strip()]
desk = [e for e in entries if e.get("Package") == "signal-desktop"]
best = max(desk, key=lambda e: [int(x) if x.isdigit() else 0 for x in e["Version"].replace("-", ".").split(".")])
print(best["Version"], best["Filename"], best["SHA256"])
EOF
read -r version filename sha < deb.txt
echo "signal-desktop $version: $filename"
[ -f "signal-desktop-$version.deb" ] || curl -fsSL -o "signal-desktop-$version.deb" "$REPO/$filename"
echo "$sha  signal-desktop-$version.deb" | sha256sum -c --quiet && echo "signal-desktop-$version.deb: SHA-256 matches Packages"
ln -sfn "signal-desktop-$version.deb" signal-desktop.deb
