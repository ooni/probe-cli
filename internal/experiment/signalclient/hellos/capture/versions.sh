#!/bin/bash
# Print the network-library versions Signal's clients currently pin, read from Signal's public
# repositories (default branches), as shell assignments for the capture scripts:
#   eval "$(./versions.sh)"
# Override any of them by setting the variable before running a capture script.
set -euo pipefail
raw() { curl -fsSL "https://raw.githubusercontent.com/signalapp/$1"; }

toml=$(raw Signal-Android/main/gradle/libs.versions.toml)
okhttp=$(grep -E '^square-okhttp3 *=' <<<"$toml" | sed -E 's/.*okhttp:([^"]+)".*/\1/')
conscrypt=$(grep -E '^conscrypt-android *=' <<<"$toml" | sed -E 's/.*conscrypt-android:([^"]+)".*/\1/')
gradle=$(raw Signal-Android/main/app/build.gradle.kts)
android_version=$(grep -m1 -E 'canonicalVersionName *= *"' <<<"$gradle" | sed -E 's/.*"([^"]+)".*/\1/')

# okio and kotlin-stdlib: whatever this OkHttp's POM depends on
pom=$(curl -fsSL "https://repo1.maven.org/maven2/com/squareup/okhttp3/okhttp-jvm/$okhttp/okhttp-jvm-$okhttp.pom")
dep() { grep -A1 "<artifactId>$1</artifactId>" <<<"$pom" | grep -m1 -o '<version>[^<]*' | cut -d'>' -f2; }

podfile=$(raw Signal-iOS/main/Podfile)
libsignal=$(grep -m1 "pod 'LibSignalClient'" <<<"$podfile" | sed -E "s/.*tag: '([^']+)'.*/\1/")
pkg=$(raw Signal-Desktop/main/package.json)
electron=$(python3 -I -c 'import json,sys; p=json.load(sys.stdin); print((p.get("devDependencies") or {}).get("electron") or p["dependencies"]["electron"])' <<<"$pkg")
desktop_version=$(python3 -I -c 'import json,sys; print(json.load(sys.stdin)["version"])' <<<"$pkg")
# The apps pin libsignal separately and not always to the same version: LIBSIGNAL_TAG follows
# Signal-iOS, LIBSIGNAL_DESKTOP_TAG Signal-Desktop; report the other pins so a difference is noticed.
desktop_libsignal=$(python3 -I -c 'import json,sys; print(json.load(sys.stdin)["dependencies"]["@signalapp/libsignal-client"])' <<<"$pkg")
catalog=$(raw Signal-Android/main/gradle/libs.versions.toml)  # read whole: grep -m1 on a pipe makes curl fail
android_libsignal=$(grep -m1 -E '^libsignal-client *=' <<<"$catalog" | sed -E 's/.*"([^"]+)".*/\1/')
for p in "Signal-Android:$android_libsignal" "Signal-Desktop:$desktop_libsignal"; do
  [ "v${p#*:}" = "$libsignal" ] || echo "versions.sh: ${p%%:*} pins libsignal ${p#*:}, not $libsignal (LIBSIGNAL_TAG follows Signal-iOS)" >&2
done

# %q: the output is eval'ed on the host, and the values come from the network.
printf '%s=%q\n' \
  LIBSIGNAL_TAG "${LIBSIGNAL_TAG:-$libsignal}" \
  LIBSIGNAL_DESKTOP_TAG "${LIBSIGNAL_DESKTOP_TAG:-v$desktop_libsignal}" \
  OKHTTP_VERSION "${OKHTTP_VERSION:-$okhttp}" \
  OKIO_VERSION "${OKIO_VERSION:-$(dep okio-jvm)}" \
  KOTLIN_VERSION "${KOTLIN_VERSION:-$(dep kotlin-stdlib)}" \
  CONSCRYPT_VERSION "${CONSCRYPT_VERSION:-$conscrypt}" \
  SIGNAL_ANDROID_VERSION "${SIGNAL_ANDROID_VERSION:-$android_version}" \
  ELECTRON_VERSION "${ELECTRON_VERSION:-$electron}" \
  SIGNAL_DESKTOP_VERSION "${SIGNAL_DESKTOP_VERSION:-$desktop_version}"
