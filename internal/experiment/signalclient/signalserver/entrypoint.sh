#!/bin/bash
# Start a Docker engine for Testcontainers, wait for it, then run the command.
set -euo pipefail
dockerd --host unix:///run/docker.sock >/var/log/dockerd.log 2>&1 &
for _ in $(seq 1 100); do docker info >/dev/null 2>&1 && break; sleep 0.3; done
docker info >/dev/null 2>&1 || { echo "dockerd did not start:" >&2; tail -20 /var/log/dockerd.log >&2; exit 1; }
exec "$@"
