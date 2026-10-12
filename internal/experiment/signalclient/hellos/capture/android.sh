#!/bin/bash
# Capture Signal-Android's two ClientHellos (provisioning WebSocket, REST/CDN) by running the
# app's OkHttp + Conscrypt configuration on the JVM (SignalAndroidHello.java), rootless, with
# pasta writing the pcap. Writes ../android-okhttp-websocket.bin and ../android-okhttp-rest.bin.
set -euo pipefail
HERE=$(cd "$(dirname "$0")" && pwd); HELLOS=$(dirname "$HERE")
WORK=${WORK_DIR:-${XDG_CACHE_HOME:-$HOME/.cache}/ooni-signal-hellos}
CTR=${CONTAINER:-podman}
: "${OKHTTP_VERSION:?run: eval \"\$(./versions.sh)\"}" "${OKIO_VERSION:?}" "${KOTLIN_VERSION:?}" "${CONSCRYPT_VERSION:?}"
# One lib directory per version set: with a shared one, a version bump would leave the old jars on
# the wildcard classpath, and the JVM could load them instead of the new ones.
LIB="lib-okhttp$OKHTTP_VERSION-okio$OKIO_VERSION-kotlin$KOTLIN_VERSION-conscrypt$CONSCRYPT_VERSION"
mkdir -p "$WORK/android/$LIB"

fetch() {  # group/path artifact version: download from Maven Central and check its sha1
  local url="https://repo1.maven.org/maven2/$1/$3/$2-$3.jar" f="$WORK/android/$LIB/$2-$3.jar"
  [ -f "$f" ] || curl -fsSL -o "$f" "$url"
  echo "$(curl -fsSL "$url.sha1" | cut -c1-40)  $f" | sha1sum -c --quiet
}
fetch com/squareup/okhttp3/okhttp-jvm okhttp-jvm "$OKHTTP_VERSION"
fetch com/squareup/okio/okio-jvm okio-jvm "$OKIO_VERSION"
fetch org/jetbrains/kotlin/kotlin-stdlib kotlin-stdlib "$KOTLIN_VERSION"
fetch org/conscrypt/conscrypt-openjdk-uber conscrypt-openjdk-uber "$CONSCRYPT_VERSION"

run() {  # name args...: run the client once under a pasta capture
  local pcap="$WORK/android/$1.pcap" log="$WORK/android/$1.log" rc=0; shift; rm -f "$pcap"
  # The client exits non-zero if the OkHttp or Conscrypt it loaded is not the version asked for.
  "$CTR" run --rm --network "pasta:--pcap,$pcap" -v "$HERE:/src:ro,Z" -v "$WORK/android:/w:Z" \
    -v "$HELLOS/signal.cer:/w/signal.cer:ro,Z" -w /w \
    -e EXPECT_OKHTTP="$OKHTTP_VERSION" -e EXPECT_CONSCRYPT="$CONSCRYPT_VERSION" \
    docker.io/library/eclipse-temurin:21-jdk \
    java -cp "/w/$LIB/*" /src/SignalAndroidHello.java "$@" >"$log" 2>&1 || rc=$?
  grep -v '^WARNING' "$log" || true
  [ "$rc" = 0 ] || { echo "android.sh: the client exited with status $rc" >&2; exit 1; }
  chmod 644 "$pcap"
}
run websocket                               # provisioning socket on chat.signal.org
run rest --get https://cdn2.signal.org/     # the same client builder for HTTPS
python3 -I "$HERE/extract_hello.py" "$WORK/android/websocket.pcap" chat.signal.org "$HELLOS/android-okhttp-websocket.bin"
python3 -I "$HERE/extract_hello.py" "$WORK/android/rest.pcap" cdn2.signal.org "$HELLOS/android-okhttp-rest.bin"
echo "Captured with OkHttp $OKHTTP_VERSION + Conscrypt $CONSCRYPT_VERSION: update Hello names in hellos.go and the table in hellos/README.md"
