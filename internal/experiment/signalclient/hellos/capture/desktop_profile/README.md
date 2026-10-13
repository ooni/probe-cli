# Profiling Signal-Desktop's traffic (experiment, 2026-10-13)

An experiment toward making the test's traffic look like a real client's, not used by the test
(ts-041 Future work). It runs Signal's official Signal-Desktop on a fresh, unlinked install, left
on its QR screen, and records which connections it opens, when, and the size and timing of its
TLS records.

```sh
./fetch.sh                                   # download and verify the current .deb (~120 MB)
./profile.sh                                 # first launch (not captured), then 240 s captured
python3 -I analyze.py "$WORK/desktop-profile/runs/<time>/capture.pcap" --records
```

`fetch.sh` checks the APT repository's `InRelease` signature with Signal's key
(`DBA36B5181D0C816F630E889D980A17457F6FB06`), then the `Packages` index and the `.deb` against the
SHA-256 values it lists. `profile.sh` runs the app rootless in podman on Xvfb, with pasta writing
the capture, and connects to Signal's production servers as a new install does. A first launch
is not captured, because it downloads a spellcheck dictionary and creates its database once.

## Results: Signal-Desktop 8.30.0, second launch, 240 s

Only `grpc.chat.signal.org`, all through libsignal 0.103.0 (its 212-byte hello, ALPN `h2`):

| Time | What |
|---|---|
| ~2.0 s | **preconnect**: TLS and the HTTP/2 preface and SETTINGS, then idle until the app quits |
| ~2.3 s | **provisioning socket**, a new connection: extended CONNECT, the server pushes the address, the client acks |
| +31 s, +62 s | a WebSocket ping (33-byte TLS record) and pong (29): libsignal's keepalive after 31 s idle. No `GET /v1/keepalive` |
| every 45 s | a new provisioning socket on a new connection (`Provisioner.loop(N): connected, refreshing in 45000ms`); earlier ones stay open |
| +90 s per socket | the server closes the WebSocket (Signal-Server's limit) |

Only sometimes:
- **First launch:** the spellcheck dictionary (~450 KB) from
  `updates.signal.org/desktop/hunspell_dictionaries/<electron>/`, through Electron's Chromium
  stack (ALPN `h2,http/1.1`) (`app/updateDefaultSession.main.ts`).
- **First launch, a new version, then at most daily:** the release-notes manifest (~66 KB) from
  `updates2.signal.org`, through Node, with the `desktop-electron` hello
  (`releaseNoteAndMegaphoneFetcher`).
- **Not while unlinked, in these runs:** the updater's `latest-linux.yml` poll.

Compared with signal_client: the provisioning flow up to the ack matches `chat_libsignal`; the
app sends no keepalive requests, only a WebSocket ping after 31 s, preconnects, and opens a new
provisioning socket every 45 s. A Desktop-shaped run would fit in 60 s (preconnect; a provisioning
socket idle until the 31 s ping; a second one at 45 s). It is future work: an app in use, with an
account, behaves differently, and differently for each user.

The screenshots `profile.sh` saves show a linking QR code; they expire, but do not share them.
