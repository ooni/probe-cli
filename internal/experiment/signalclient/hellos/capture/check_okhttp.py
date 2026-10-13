#!/usr/bin/env python3
"""Check that OkHttp still configures Conscrypt sockets on the JVM as it does on Android.

android.sh captures Signal-Android's hellos on the JVM with Conscrypt as the first security
provider, so that OkHttp's JVM build uses ConscryptPlatform. That platform's configureTlsExtensions
makes the same calls as the Android build's ConscryptSocketAdapter, which a phone uses for Signal's
bundled Conscrypt (session tickets on, then ALPN). The two produced byte-identical hellos when
checked against Android 14's own runtime (hellos/README.md). This compares both functions in the
given OkHttp version's sources with the copies in okhttp-conscrypt-tls.txt, and fails if either
changed, so that the equivalence is checked again before a capture is trusted.

Usage: check_okhttp.py OKHTTP_VERSION CACHE_DIR [--write]   (--write: record the current sources)
"""
import hashlib
import io
import os
import re
import sys
import urllib.request
import zipfile

HERE = os.path.dirname(os.path.abspath(__file__))
REFERENCE = os.path.join(HERE, "okhttp-conscrypt-tls.txt")
SOURCES = [  # (artifact, path in its sources jar)
    ("okhttp-android", "androidMain/okhttp3/internal/platform/android/ConscryptSocketAdapter.kt"),
    ("okhttp-jvm", "jvmMain/okhttp3/internal/platform/ConscryptPlatform.kt"),
]


def sources_jar(artifact, version, cache):
    url = f"https://repo1.maven.org/maven2/com/squareup/okhttp3/{artifact}/{version}/{artifact}-{version}-sources.jar"
    path = os.path.join(cache, f"{artifact}-{version}-sources.jar")
    if not os.path.exists(path):
        with urllib.request.urlopen(url) as r, open(path, "wb") as f:
            f.write(r.read())
    with urllib.request.urlopen(url + ".sha1") as r:
        want = r.read().decode().split()[0]
    data = open(path, "rb").read()
    if hashlib.sha1(data).hexdigest() != want:
        sys.exit(f"check_okhttp.py: {path} does not match Maven's SHA-1")
    return zipfile.ZipFile(io.BytesIO(data))


def function(source, name):
    """The text of `override fun name(...) { ... }`, whitespace-normalized."""
    m = re.search(r"override fun " + name + r"\(", source)
    if not m:
        return None
    i = source.index("{", m.end())
    depth, j = 0, i
    while True:
        depth += {"{": 1, "}": -1}.get(source[j], 0)
        j += 1
        if depth == 0:
            break
    return " ".join(source[m.start():j].split())


def main():
    version, cache = sys.argv[1], sys.argv[2]
    current = []
    for artifact, path in SOURCES:
        text = sources_jar(artifact, version, cache).read(path).decode()
        body = function(text, "configureTlsExtensions")
        if body is None:
            sys.exit(f"check_okhttp.py: no configureTlsExtensions in {artifact} {path}")
        current.append(f"{artifact} {path}\n{body}\n")
    current = "".join(current)
    if "--write" in sys.argv:
        with open(REFERENCE, "w") as f:
            f.write(f"# OkHttp {version}: verified against Android 14's runtime (hellos/README.md)\n" + current)
        print(f"check_okhttp.py: recorded OkHttp {version}")
        return
    reference = "".join(l for l in open(REFERENCE) if not l.startswith("#"))
    if reference != current:
        sys.exit(f"check_okhttp.py: OkHttp {version} changed how it configures Conscrypt sockets. The JVM "
                 "capture may no longer match a phone: check it against Android's runtime (hellos/README.md), "
                 "then run check_okhttp.py --write.")
    print(f"check_okhttp.py: OkHttp {version} configures Conscrypt sockets as verified")


main()
