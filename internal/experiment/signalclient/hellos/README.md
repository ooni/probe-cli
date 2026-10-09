# Captured Signal ClientHellos

Each `.bin` file is a TLS ClientHello handshake message (starting with type 1, without a record
header), captured from the client's own network code from an unfiltered network on 2026-10-09.
`hellos.go` turns them into uTLS specs.

| File | Client | How captured | JA4 |
|---|---|---|---|
| `libsignal-0.105.0.bin` | libsignal 0.105.0 direct route to grpc.chat.signal.org (pinned by Signal-Android, Signal-iOS and Signal-Desktop) | libsignal's own `chat_smoke_test` example (`rust/net/examples`), tcpdump | `t13d0307h2_55b375c5d22e_3df4dd8a7a2a` |
| `android-okhttp-websocket.bin` | Signal-Android 8.29.3 provisioning WebSocket to chat.signal.org | the app's OkHttp 5.3.2 + Conscrypt 2.6.2 configuration rebuilt on the JVM (`SignalUrlExtensions.buildOkHttpClient`, `Tls12SocketFactory`, `ProvisioningSocket`); Conscrypt openjdk-uber 2.6.2, the same version the app bundles | `t13d0911h1_f91f431d341e_86dd91ae2a36` |
| `android-okhttp-rest.bin` | the same client, HTTPS GET to cdn2.signal.org | as above | `t13d0911h2_f91f431d341e_86dd91ae2a36` |

**Derived, not captured:** libsignal's enclave connections (CDSI, SVR2, SVR-B) use the same
`ssl_config` with ALPN `http/1.1`, so `HelloLibsignalEnclave` is the libsignal hello with its
ALPN replaced.

**Not yet available:** iOS `URLSession` (Apple's TLS stack, for CDNs and storage) needs an iPhone
or a Mac. Signal-Desktop's own fetches (Electron) need a Desktop capture.

The roots are Signal's RSA root (`signal.cer`) and its Ed25519 root (`signal-ed25519.cer`),
copied from libsignal `rust/net/res/`.

**Refreshing:** re-capture whenever the pinned libsignal, OkHttp or Conscrypt version changes, and
update this table.
