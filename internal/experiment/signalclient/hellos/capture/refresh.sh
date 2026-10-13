#!/bin/bash
# Re-capture the ClientHellos that signal_client reproduces, from the clients' own code at the
# versions Signal currently pins.   Usage: ./refresh.sh [libsignal] [android] [desktop]   (default: all)
# Requirements: podman (its pasta network writes the captures), curl, git, python3 (for versions.sh
# and desktop.sh). hellotool is built from this checkout, with Go or in a golang container (../../gotool.sh).
# Afterwards: go test ./internal/experiment/signalclient/ (the tests compare generated and
# captured hellos), update names in hellos.go and the table in hellos/README.md, and commit.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
for c in "$@"; do
  case $c in libsignal|android|desktop) ;; *) echo "usage: refresh.sh [libsignal] [android] [desktop]" >&2; exit 2 ;; esac
done
versions=$("$HERE/versions.sh")  # an assignment, so that set -e stops on its failure
echo "$versions"
eval "$versions"
export LIBSIGNAL_TAG LIBSIGNAL_DESKTOP_TAG OKHTTP_VERSION OKIO_VERSION KOTLIN_VERSION CONSCRYPT_VERSION ELECTRON_VERSION
for c in ${@:-libsignal android desktop}; do
  echo "== $c"; "$HERE/$c.sh"
done
git -C "$HERE" status --short -- ..
