#!/bin/bash
# Re-capture the ClientHellos that signal_client reproduces, from the clients' own code at the
# versions Signal currently pins.   Usage: ./refresh.sh [libsignal] [android] [desktop]   (default: all)
# Requirements: podman (or CONTAINER=docker with a pasta-capable setup), curl, git, python3.
# Afterwards: go test ./internal/experiment/signalclient/ (the tests compare generated and
# captured hellos), update names in hellos.go and the table in hellos/README.md, and commit.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
eval "$("$HERE/versions.sh")"
export LIBSIGNAL_TAG OKHTTP_VERSION OKIO_VERSION KOTLIN_VERSION CONSCRYPT_VERSION ELECTRON_VERSION
"$HERE/versions.sh"
for c in ${@:-libsignal android desktop}; do
  echo "== $c"; "$HERE/$c.sh"
done
git -C "$HERE" status --short -- ..
