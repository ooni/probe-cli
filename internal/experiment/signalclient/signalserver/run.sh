#!/bin/bash
# Run Signal-Server's own feature-limited test mode (LocalWhisperServerService: stubbed registration,
# with Redis, DynamoDB Local and FoundationDB started by Testcontainers) with TLS, so signal_client's
# real handshakes and flows can be checked against Signal's server code. Opt-in tooling: probe-cli
# never needs it. Signal-Server is used unmodified; only its config is changed.
#
#   127.0.0.1:8443  chat.signal.org: a Jetty TLS connector (h2 + http/1.1), set to production's profile.
#   127.0.0.1:8444  grpc.chat.signal.org: Signal-Server's own Netty front end (OmnibusH2Server: TLS 1.3,
#                   h2 only, BoringSSL), which proxies /v1/websocket/ to Jetty's internal h2c port 8080.
#
# Testcontainers needs a container engine: the server runs in an image with a Docker engine
# (Containerfile), with --privileged (for rootless podman, privileges only inside the user
# namespace). Nested images are kept in the signal-server-docker volume.
#
# Usage: run.sh        (then ./test.sh, then ./stop.sh)
# Env: SIGNAL_SERVER_REF (branch or commit, default main), SIGNAL_SERVER_DIR (existing checkout, used as is),
#      WORK_DIR (default ~/.cache/ooni-signal-server), CONTAINER (default podman)
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd)
CTR=${CONTAINER:-podman}
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-server}; mkdir -p "$WORK/certs"
SRC=${SIGNAL_SERVER_DIR:-$WORK/Signal-Server}
JDK=docker.io/library/eclipse-temurin:26-jdk   # Signal-Server's .java-version: temurin-26
SERVERTOOL=$("$HERE/../gotool.sh" --path servertool)  # built from this checkout
# Tracks main by default, so changes in Signal-Server that break the harness show up. Last known to
# work with a15d5cdbb937fc5d078b73ec7b4a725ddf4e9473 (2026-10-07): pin it with SIGNAL_SERVER_REF if needed.
if [ -z "${SIGNAL_SERVER_DIR:-}" ]; then
  [ -d "$SRC/.git" ] || git init -q "$SRC"
  git -C "$SRC" fetch -q --depth 1 https://github.com/signalapp/Signal-Server.git "${SIGNAL_SERVER_REF:-main}"
  git -C "$SRC" -c advice.detachedHead=false checkout -q --force FETCH_HEAD
fi
echo "Signal-Server $(git -C "$SRC" log -1 --format='%h %cs' HEAD) ($SRC)"
"$CTR" volume exists signal-server-m2 || "$CTR" volume create signal-server-m2 >/dev/null

# 0. Maven at the version and SHA-256 Signal-Server pins (.mvn/wrapper/maven-wrapper.properties).
# We fetch and verify it ourselves: the wrapper script's own download check fails in the slim JDK image.
props="$SRC/.mvn/wrapper/maven-wrapper.properties"
murl=$(sed -n 's/^distributionUrl=//p' "$props"); msha=$(sed -n 's/^distributionSha256Sum=//p' "$props")
MVN_DIR="$WORK/maven-$msha"
if [ ! -x "$MVN_DIR/bin/mvn" ]; then
  curl -fsSL -o "$WORK/maven.zip" "$murl"
  echo "$msha  $WORK/maven.zip" | sha256sum -c --quiet
  "$SERVERTOOL" unpack-maven "$WORK/maven.zip" "$MVN_DIR"
  rm -f "$WORK/maven.zip"
fi

# 0b. FoundationDB's native client library, at the version and SHA-256 Signal-Server's pom pins (its
#     Maven build downloads it only when packaging the server image).
fdbver=$(sed -n 's|.*<foundationdb.version>\(.*\)</foundationdb.version>.*|\1|p' "$SRC/pom.xml")
fdbsha=$(sed -n 's|.*<foundationdb.client-library-sha256>\(.*\)</foundationdb.client-library-sha256>.*|\1|p' "$SRC/pom.xml")
[ -n "$fdbver" ] && [ -n "$fdbsha" ] || { echo "no FoundationDB version or checksum in Signal-Server's pom: update signalserver/run.sh" >&2; exit 1; }
FDB_DIR="$WORK/fdb-$fdbver"
if [ ! -f "$FDB_DIR/libfdb_c.so" ]; then
  mkdir -p "$FDB_DIR"
  curl -fsSL -o "$FDB_DIR/libfdb_c.so.tmp" "https://github.com/apple/foundationdb/releases/download/$fdbver/libfdb_c.x86_64.so"
  echo "$fdbsha  $FDB_DIR/libfdb_c.so.tmp" | sha256sum -c --quiet
  mv "$FDB_DIR/libfdb_c.so.tmp" "$FDB_DIR/libfdb_c.so"
fi

# 1. test certificates shaped like Signal's production chain (servertool certs), and the connectors' PKCS#12
#    keystores (MakeKeyStores.java). Valid 30 days: regenerated when missing, older than 25 days, or
#    made with another password. Signal-Server overrides each connector's keystore password with its
#    tlsKeyStore.password secret, so the test secret is used.
KSPASS=$(sed -n 's/^tlsKeyStore\.password: *//p' "$SRC/service/src/test/resources/config/test-secrets-bundle.yml")
[ -n "$KSPASS" ] || { echo "no tlsKeyStore.password in Signal-Server's test secrets: update signalserver/run.sh" >&2; exit 1; }
C="$WORK/certs"
if [ ! -f "$C/chat.p12" ] || [ ! -f "$C/grpc.p12" ] || [ ! -f "$C/ca.pem" ] ||
   [ "$(cat "$C/password" 2>/dev/null)" != "$KSPASS" ] || [ -n "$(find "$C/chat.p12" -mtime +25)" ]; then
  rm -f "$C"/*
  "$SERVERTOOL" certs "$C"
  "$CTR" run --rm -e KSPASS="$KSPASS" -v "$HERE:/h:ro,Z" -v "$C:/certs:Z" "$JDK" \
    sh -ec 'java /h/MakeKeyStores.java /certs "$KSPASS"; rm -f /certs/*.key'
  printf '%s' "$KSPASS" > "$C/password"
fi

# 2. the test config: a TLS connector for chat next to the original h2c one, and the gRPC front end with TLS
"$SERVERTOOL" config "$SRC/service/src/test/resources/config/test.yml" "$WORK/test-tls.yml" "$KSPASS"

# 3. the image (JDK + Docker engine) and the nested image store
IMG=localhost/signal-server-run
"$CTR" build -q -t "$IMG" "$HERE" >/dev/null
"$CTR" volume exists signal-server-docker || "$CTR" volume create signal-server-docker >/dev/null

# 4. build (once) and run
"$CTR" rm -f signal-server-local >/dev/null 2>&1 || true
"$CTR" run -d --name signal-server-local --privileged --memory 8g --memory-swap 8g -p 127.0.0.1:8443:8443 -p 127.0.0.1:8444:8444 \
  -v "$SRC:/src:Z" -v "$C:/certs:ro,Z" -v "$WORK/test-tls.yml:/test-tls.yml:ro,Z" \
  -v "$MVN_DIR:/maven:ro,Z" -v signal-server-m2:/root/.m2 -v signal-server-docker:/var/lib/docker \
  -v "$FDB_DIR:/opt/fdb:ro,Z" -e LD_LIBRARY_PATH=/opt/fdb \
  -e SIGNAL_SERVER_CONFIG=/test-tls.yml -w /src "$IMG" \
  /maven/bin/mvn -B -q -DskipTests=true -Ptest-server integration-test >/dev/null
echo "starting (first run builds and pulls images; follow with: $CTR logs -f signal-server-local)"
for _ in $(seq 1 180); do
  if [ "$("$CTR" inspect -f '{{.State.Running}}' signal-server-local 2>/dev/null)" != true ]; then
    echo "signal-server-local exited:" >&2; "$CTR" logs --tail 40 signal-server-local >&2; exit 1
  fi
  if "$CTR" exec signal-server-local bash -c 'exec 3<>/dev/tcp/127.0.0.1/8443 && exec 4<>/dev/tcp/127.0.0.1/8444' 2>/dev/null; then
    echo "Signal-Server test mode listening: chat.signal.org profile on 127.0.0.1:8443, grpc.chat.signal.org profile on 127.0.0.1:8444. Now: $HERE/test.sh"; exit 0
  fi
  sleep 5
done
echo "not listening after 15 min; see: $CTR logs signal-server-local" >&2; exit 1
