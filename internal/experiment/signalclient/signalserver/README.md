# Testing signal_client against Signal-Server's own code (opt-in)

These scripts run Signal-Server's feature-limited test mode (`LocalWhisperServerService`) locally,
configured so that its TLS and HTTP/2 front ends look like production's `chat.signal.org` and
`grpc.chat.signal.org`, and run signal_client's real handshakes and provisioning flows against it.
probe-cli never needs any of this: the Go test is behind the `signalserver` build tag.

```sh
./run.sh        # build (first run: several minutes) and start; needs podman
./test.sh       # go test -tags signalserver against it
./tlsprofile.sh # measure production and the local server; compare tlsprofile-{prod,local}.json
./stop.sh
```

## What runs

- **Signal-Server**, from a checkout (`SIGNAL_SERVER_DIR`, or a clone of `SIGNAL_SERVER_REF`),
  built with the Maven version and SHA-256 its wrapper pins, on the JDK its `.java-version` names
  (Temurin 26; `JDK` in run.sh, updated by hand).
- **Its dependencies**, started by Signal-Server's own test code through Testcontainers: a
  three-node Redis cluster (Docker Compose), DynamoDB Local and FoundationDB. They run in a Docker
  engine inside the server container (`Containerfile`), which runs `--privileged`; with rootless
  podman that grants privileges only inside your user namespace. Docker rather than podman inside,
  because Testcontainers' Compose support uses container links, which podman's API rejects.
- **FoundationDB's native client** (`libfdb_c.so`), which its Java bindings load: Apple's release
  binary at the version and SHA-256 Signal-Server's pom pins (open source, Apache 2.0).
- **Two front ends**, configured from measurements of production (`tlsprofile.sh`, 2026-10):
  - `127.0.0.1:8443`, shaped like `chat.signal.org`: a Jetty TLS connector (TLS 1.2 and 1.3,
    production's cipher suites in its order, an Ed25519 and an RSA certificate chosen by the
    client's signature algorithms);
  - `127.0.0.1:8444`, shaped like `grpc.chat.signal.org`: Signal-Server's own gRPC front end with
    TLS (TLS 1.3 only, X25519MLKEM768, h2 only, 100 concurrent streams), proxying WebSockets to the
    server's h2c connector.
- **Certificates** (`servertool certs`, `MakeKeyStores.java`): local test CAs and leaves with the same
  shape as production's (key types, signature algorithms, empty subject, extensions in production's
  order),
  named as test CAs. The keystores use Signal-Server's test secret as their password because the
  server overrides connector keystore passwords with its `tlsKeyStore.password` secret.

Downloads and caches live in `~/.cache/ooni-signal-server` (`WORK_DIR`) and in the podman volumes
`signal-server-m2` and `signal-server-docker`. The helpers in `cmd/servertool` (certificates,
test configuration, Maven's unpacking, the TLS profile) are built by `../gotool.sh`.

## Keeping it matching production

Run `./tlsprofile.sh` after Signal changes its servers: it records TLS versions, each cipher suite
and the server's preference, key-exchange groups, the certificate chain, ALPN, session tickets,
OCSP stapling and the HTTP/2 SETTINGS for both hostnames, in production and locally (about 100
ordinary handshakes per hostname). Update `run.sh` until the two JSON files agree. On 2026-10-13
they differed only in the certificates' issuers and in session tickets: the local front ends send
two after a TLS 1.3 handshake, production none.

Known gaps: only the two chat hostnames are modelled (not the CDNs or enclaves), and the local
server's data and behaviour are Signal-Server's test mode, not production's.
