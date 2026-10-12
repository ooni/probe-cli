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
| `android-okhttp-websocket.bin` | Signal-Android (OkHttp 5.3.2 + Conscrypt 2.6.2), provisioning WebSocket to chat.signal.org | the app's client configuration (`SignalUrlExtensions.buildOkHttpClient`, `Tls12SocketFactory`, `ProvisioningSocket`) rebuilt on the JVM in `capture/SignalAndroidHello.java`, with the same library versions; Conscrypt's openjdk-uber build | `t13d0911h1_f91f431d341e_86dd91ae2a36` |
| `android-okhttp-rest.bin` | the same client, HTTPS to storage and the CDNs (ALPN h2,http/1.1) | as above, a GET to cdn2.signal.org | `t13d0911h2_f91f431d341e_86dd91ae2a36` |
| `libsignal-reflector.bin` | libsignal v0.105.0's connection to a front on the reflector routes (proxyf, proxyg): TLS 1.2 and 1.3, ALPN http/1.1, padded to 512 bytes. Captured on all eight fronts; `same_hello.py` checks they are one hello | `chat_smoke_test --limit-to-routes proxyf` and `proxyg` with `--try-all-routes` | `t13d1712h1_5b57614c22b0_f014b29961f9` |
| `android-okhttp-front-g-gmail-websocket.bin`, `…-rest.bin` | Signal-Android's Google fronts with `GMAIL_CONNECTION_SPEC` (also `PLAY_CONNECTION_SPEC`, the same list): www.google.com, googlemail.com, android.clients.google.com and the countries' Google domains. TLS 1.2 only, 9 suites, empty session id | `SignalAndroidHello.java --spec gmail`, provisioning through `www.google.com` and a storage GET, with the relay's Host | `t12d0908h1_5e39edccbee7_036209cd1ead`, `t12d0908h2_…` |
| `android-okhttp-front-g-gmaps-websocket.bin`, `…-rest.bin` | the same with `GMAPS_CONNECTION_SPEC`, for clients3 and clients4.google.com. TLS 1.2 only, 14 suites, empty session id | `--spec gmaps` through `clients3.google.com` | `t12d1408h1_c866b44c5a26_036209cd1ead`, `t12d1408h2_…` |
| `android-okhttp-front-f-websocket.bin`, `…-rest.bin` | Signal-Android's Fastly fronts with `ConnectionSpec.MODERN_TLS` | `--spec modern` through `github.githubassets.com` | `t13d1511h1_8daaf6152771_86dd91ae2a36`, `t13d1511h2_…` |
| `desktop-electron.bin` | Signal-Desktop's own fetches (updater, WebAPI agent) through Electron 44.4.5 (Node + BoringSSL): no ALPN, X25519MLKEM768 + x25519 key shares, session_ticket | `capture/desktop-fetch.js` under the pinned Electron, with `createHTTPSAgent`'s TLS options (Signal's RSA root as `ca`) | `t13d181000_5d04281c6031_78e6aca7449b` |

**Derived, not captured:** libsignal's enclave connections (CDSI, SVR2, SVR-B) use the same
`ssl_config` with ALPN `http/1.1` (`rust/net/src/env.rs`), so `HelloLibsignalEnclave` is the
libsignal hello with its ALPN replaced.

**Session id:** uTLS always sends a 32-byte legacy session id. The TLS 1.2-only Google-front
hellos send an empty one, as OpenSSL and BoringSSL do without TLS 1.3, so `Hello.apply` empties it
for hellos captured that way, and the tests compare its length.

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

`capture/extract_hello.py` turns a capture into the `.bin` files, so it is tested against known
values two ways:

```sh
cd capture
python3 -m unittest test_extract_hello   # offline, seconds
./check_extract.py                       # real connections; needs podman with pasta and network
```

- `test_extract_hello.py` writes the committed hellos (with fresh random values) into synthetic
  captures and expects them back byte for byte. The captures cover reordered, retransmitted and
  overlapping segments; flows without their SYN; sequence numbers wrapping past 2^32; Ethernet
  padding and VLAN tags; every supported link type, over IPv4 and IPv6; and pcap and pcapng.
- `check_extract.py` uses ground truth. `openssl s_client -msg` prints the ClientHello it sent,
  pasta captures the same connection, and the extractor must agree with OpenSSL. Cases cover
  hellos split over two and three TCP segments, a hello in four TLS records, a HelloRetryRequest,
  and decoy connections. Each case also checks that it exercised what it is for.

Run both after changing the extractor.
