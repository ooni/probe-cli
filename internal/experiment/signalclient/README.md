# signal_client

Measures whether Signal can be used, and how it is blocked, by connecting the way Signal's own
clients do. The specification is ooni/spec `nettests/ts-041-signal-client.md`; `doc.go` maps the
files. This page covers the options and the choices the spec leaves to the implementation.

## Options

Options are given with `-O` (miniooni), in an OONI Run v2 descriptor, or by the backend. Lists are
written as strings. An invalid option fails the run before anything is measured.

| Option | Default | Effect |
|---|---|---|
| `IdleSeconds` | `0` | idle time on provisioning sockets, capped at 60 |
| `AppUserAgent` | `false` | send the Signal apps' User-Agents instead of identifying as OONI Probe |
| `UserAgent` | empty | User-Agent sent instead of either |
| `SignalCA` | empty | PEM bundle that replaces Signal's two roots (targets, verifying controls, inner connections of the routes; not the fronts or DoH). Recorded in `signal_roots` |
| `Controls` | `on_failure` | `on_failure`, `always` or `off` |
| `StaticAddresses` | empty | `host=addr,addr;host=addr`: replaces the named hosts' static addresses |
| `Circumvention` | empty | comma-separated `reflector_f`, `reflector_g`, `android_g`, `android_f`, or `all` |
| `CircumventionCountry` | probe's country | the country whose Android front configuration is reproduced |
| `Fronts` | empty | `route[.service]=host_header@sni,sni;...`: replaces the named routes' fronts |

Examples:

```
miniooni -n signal_client -O Controls=always
miniooni -n signal_client -O Circumvention=all -O CircumventionCountry=IR
miniooni -n signal_client -O 'StaticAddresses=chat.signal.org=13.248.212.111,76.223.92.165'
miniooni -n signal_client -O Circumvention=reflector_f \
    -O 'Fronts=reflector_f=reflector-signal.global.ssl.fastly.net@github.githubassets.com'
```

The test is not enabled by default: set `OONI_FORCE_ENABLE_EXPERIMENT=1` to run it before the
`signal_client_enabled` check-in flag is on.

## Choices the spec leaves open

- **`always`:** every control kind runs for every target, as if it had failed at the TLS
  handshake. Handshake controls go to the target's own address, or to the first static address
  when resolution failed.
- **`address_source`:** an address listed by several clients is attributed to the first list
  that has it, in the order libsignal, Signal-Android, Signal-Desktop. For example, chat.signal.org's
  addresses show `android`, although Desktop lists them too.
- **`StaticAddresses`:** an entry replaces the whole list of its host, including addresses from
  other clients. Hosts that are not targets are accepted and ignored.
- **DoH fallback:** the IPv6 resolver is tried only when the IPv4 one fails without a DNS answer.
  Any `dns_*` failure (for example NXDOMAIN) is an answer. Each query has its own 10 s limit, and
  `resolver` names the resolver of the recorded answer.
- **`tls12`:** the hello offers the target's ALPN, and its name says so:
  `go-crypto-tls-1.2+h2`, `go-crypto-tls-1.2+h2,http/1.1`, or `go-crypto-tls-1.2` for none.
- **`browser_hello`:** uTLS's current Chrome parrot, named `utls-chrome-<version>`.
- **`Fronts`:**
  - An Android route without a service (`android_g=...`) sets both its services.
  - A service whose SNIs leave out the route's front resolves and dials its own front.
- **Front choice:**
  - The reflectors pick a front at random, as libsignal does.
  - `android_f` picks at random for IR and CU (Signal-Android's Fastly countries), otherwise it uses
    the first front.
  - `android_g` uses the first front, which is the country's own Google domain when it has one.
- **Front hellos:** each front connection sends the hello Signal's client sends to that front,
  captured from the client's own code (`hellos/README.md`) and recorded by name:
  - reflectors: `libsignal-<v>-reflector`, the same on all eight fronts;
  - `android_g`: Gmail's settings (`-gmail`), or Maps' (`-gmaps`) on `clients3` and
    `clients4.google.com`, both TLS 1.2 only with an empty session id, as Signal-Android sends;
  - `android_f`: OkHttp's `MODERN_TLS` (`-modern`).

  Android chat uses the WebSocket hello (`-websocket`, ALPN `http/1.1`), storage the HTTPS one
  (`h2,http/1.1`). A front given with `Fronts` that the client does not use gets the route's
  default hello (Gmail's on `android_g`).
- **Connect budget through a route:** each target reached through a route has its own 30 s
  connect budget, as a direct target does. It covers the TCP and TLS to the front, on the
  reflectors the tunnel upgrade and the inner handshake, and then the upgrade or GET. The front's
  DNS lookup has its own 30 s, and the provisioning steps keep their usual limits.
- **Route `host_header`, `front` and `front_client_hello`** describe the route's first
  connection. `tls_handshakes` records each connection's SNI.
- **The User-Agent on the reflector tunnel's upgrade** is libsignal's, as on libsignal's routes:
  the OONI one with ` libsignal/<version>` by default, the app's with `AppUserAgent`. libsignal
  sends its connection manager's User-Agent on the tunnel too
  (`rust/net/infra/src/route/proxy.rs`).

## Tests

- `go test -short ./internal/experiment/signalclient/` runs the unit tests and the netemx QA
  scenarios (offline).
- `signalserver/` runs the provisioning flow against Signal-Server's own code (opt-in; see its
  README).
- `hellos/README.md` explains how the ClientHellos are captured and refreshed.

The QA network creates only the simulated hosts the scenarios need. ooni/netem before the fix in
`fix-gvisor-stack-close` does not release a host's gVisor stack when an env closes.
