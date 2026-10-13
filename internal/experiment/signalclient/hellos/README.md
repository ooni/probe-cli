# Captured Signal ClientHellos

Each `.bin` file is a TLS ClientHello handshake message (starting with type 1, without a record
header), captured from the client's own network code from an unfiltered network. `hellos.go` turns
each into a uTLS spec.

- **Normalized:** the per-connection random values (client random, legacy session id, key share
  public keys) are zeroed, keeping their lengths. uTLS generates fresh ones, and the files change
  only when the handshake does.
- **Tests:** `hellos_test.go` checks that uTLS reproduces each hello's cipher suites, extension
  order, groups, signature algorithms, ALPN and size.

| File | Client (version captured) | How | JA4 |
|---|---|---|---|
| `libsignal.bin` | libsignal v0.105.0, direct route to grpc.chat.signal.org. Pinned by Signal-Android and Signal-iOS (chat on both, provisioning on iOS). Signal-Desktop pins v0.103.0 (chat and provisioning): captured separately (`libsignal.sh desktop`), it sends the same hello byte for byte (2026-10-12). Desktop never enables libsignal's censorship circumvention, so it has no reflector hello | libsignal's own `chat_smoke_test` example | `t13d0307h2_55b375c5d22e_3df4dd8a7a2a` |
| `android-okhttp-websocket.bin` | Signal-Android (OkHttp 5.3.2 + Conscrypt 2.6.2), provisioning WebSocket to chat.signal.org | the app's client configuration (`SignalUrlExtensions.buildOkHttpClient`, `Tls12SocketFactory`, `ProvisioningSocket`) rebuilt on the JVM in `capture/SignalAndroidHello.java`, with the same library versions; Conscrypt's openjdk-uber build, as the first security provider so that OkHttp sets sockets up as on a phone (see below) | `t13d0912h1_f91f431d341e_40271e0a5736` |
| `android-okhttp-rest.bin` | the same client, HTTPS to storage and the CDNs (ALPN h2,http/1.1) | as above, a GET to cdn2.signal.org | `t13d0912h2_f91f431d341e_40271e0a5736` |
| `libsignal-reflector.bin` | libsignal v0.105.0's connection to a front on the reflector routes (proxyf, proxyg): TLS 1.2 and 1.3, ALPN http/1.1, padded to 512 bytes. Captured on all eight fronts; `hellotool same` checks they are one hello | `chat_smoke_test --limit-to-routes proxyf` and `proxyg` with `--try-all-routes` | `t13d1712h1_5b57614c22b0_f014b29961f9` |
| `android-okhttp-front-g-gmail-websocket.bin`, `…-rest.bin` | Signal-Android's Google fronts with `GMAIL_CONNECTION_SPEC` (also `PLAY_CONNECTION_SPEC`, the same list): www.google.com, googlemail.com, android.clients.google.com and the countries' Google domains. TLS 1.2 only, 9 suites, empty session id | `SignalAndroidHello.java --spec gmail`, provisioning through `www.google.com` and a storage GET, with the relay's Host | `t12d0909h1_5e39edccbee7_b39be8c56a14`, `t12d0909h2_…` |
| `android-okhttp-front-g-gmaps-websocket.bin`, `…-rest.bin` | the same with `GMAPS_CONNECTION_SPEC`, for clients3 and clients4.google.com. TLS 1.2 only, 14 suites, empty session id | `--spec gmaps` through `clients3.google.com` | `t12d1409h1_c866b44c5a26_b39be8c56a14`, `t12d1409h2_…` |
| `android-okhttp-front-f-websocket.bin`, `…-rest.bin` | Signal-Android's Fastly fronts with `ConnectionSpec.MODERN_TLS` | `--spec modern` through `github.githubassets.com` | `t13d1512h1_8daaf6152771_40271e0a5736`, `t13d1512h2_…` |
| `desktop-electron.bin` | Signal-Desktop's own fetches (updater, WebAPI agent) through Electron 44.4.5 (Node + BoringSSL): no ALPN, X25519MLKEM768 + x25519 key shares, session_ticket | `capture/desktop-fetch.js` under the pinned Electron, with `createHTTPSAgent`'s TLS options (Signal's RSA root as `ca`) | `t13d181000_5d04281c6031_78e6aca7449b` |

**Derived, not captured:** libsignal's enclave connections (CDSI, SVR2, SVR-B) use the same
`ssl_config` with ALPN `http/1.1` (`rust/net/src/env.rs`), so `HelloLibsignalEnclave` is the
libsignal hello with its ALPN replaced.

**Session id:** uTLS always sends a 32-byte legacy session id. The TLS 1.2-only Google-front
hellos send an empty one, as OpenSSL and BoringSSL do without TLS 1.3, so `Hello.apply` empties it
for hellos captured that way, and the tests compare its length.

**Android hellos on the JVM, checked against Android's runtime:** on a phone, OkHttp's Android
build uses `Android10Platform`, whose `ConscryptSocketAdapter` turns session tickets on for the
app's bundled Conscrypt sockets, then sets ALPN. On the JVM, OkHttp does the same only through
`ConscryptPlatform`, which it picks when Conscrypt is the first security provider; with the app's
order (Conscrypt second) it uses `Jdk9Platform`, and the hello lacks the session_ticket extension.
So `android.sh` puts Conscrypt first (`CONSCRYPT_FIRST=1`) and refuses to capture unless OkHttp
reports `ConscryptPlatform`. On 2026-10-13 the REST hello and the Gmail-front REST hello were also
captured with the same client running on Android 14's own runtime (Redroid 14's system image run
rootless without init: ART via `app_process64`, `okhttp-android` and `conscrypt-android` 2.6.2 with
the app's provider order): both were byte-identical to the JVM captures. `hellotool check-okhttp` stops a
refresh if OkHttp changes either code path (`okhttp-conscrypt-tls.txt`), so the check is repeated
before a capture is trusted.

**Not available:** iOS `URLSession` (Apple's TLS stack, used by Signal-iOS for the CDNs and
storage) can only be captured on an iPhone or a Mac. Desktop's hello comes from Electron, not the
packaged app; the TLS stack and options are the same.

**Roots:** Signal's RSA root (`signal.cer`) and Ed25519 root (`signal-ed25519.cer`), copied from
libsignal `rust/net/res/`.

## Refreshing

When Signal bumps libsignal, OkHttp, Conscrypt or Electron:

```
cd capture
./versions.sh                 # versions Signal currently pins, read from its repositories
./refresh.sh                  # or any subset: ./refresh.sh libsignal android desktop
go test ./internal/experiment/signalclient/
```

The scripts run the real client code in rootless containers (podman; pasta writes the pcaps) and
write the `.bin` files here. If a file changes:
1. update the `Hello` name in `hellos.go` with the new version, and its `appUserAgent` (the app
   User-Agent sent with `AppUserAgent`);
2. update this table, including the new JA4 printed by the scripts;
3. bump the experiment version.

**Cost:** `libsignal.sh` builds libsignal's Rust example, which takes several minutes and up to
8 GB of RAM (`BUILD_MEMORY`). The others take seconds. Downloads and build caches go in
`~/.cache/ooni-signal-hellos` (`WORK_DIR`), outside the source tree.

## Checking the extractor

`capture/cmd/hellotool` turns a capture into the `.bin` files (`../gotool.sh hellotool extract`
builds and runs it), so it is tested against known values two ways:

```sh
go test ./internal/experiment/signalclient/hellos/capture/cmd/hellotool   # offline, seconds
internal/experiment/signalclient/gotool.sh hellotool check-extract       # real connections; podman with pasta
```

- The Go tests write the committed hellos (with fresh random values) into synthetic captures,
  built with gopacket, and expect them back byte for byte. The captures cover reordered,
  retransmitted and overlapping segments; flows without their SYN; sequence numbers wrapping past
  2^32; Ethernet padding; IPv4 and IPv6; and pcap and pcapng. The extractor reads Ethernet
  captures only, as pasta writes them.
- `check-extract` uses ground truth. `openssl s_client -msg` prints the ClientHello it sent,
  pasta captures the same connection, and the extractor must agree with OpenSSL. Cases cover
  hellos split over two and three TCP segments, a hello in four TLS records, a HelloRetryRequest,
  and decoy connections. Each case also checks that it exercised what it is for.

Run both after changing the extractor.
