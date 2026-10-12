// Package signalclient contains the signal_client experiment (provisional name: the final name is the
// maintainers' call). It measures whether Signal is reachable by behaving like real Signal
// clients, so that censorship keyed on how Signal looks on the wire is measured:
//
//   - every TLS handshake uses the ClientHello the real client sends, byte for byte (hellos/);
//   - the chat hosts are exercised with the flow of a freshly installed client that has no
//     account yet: the provisioning WebSocket (libsignal's HTTP/2 extended CONNECT on
//     grpc.chat.signal.org, Android OkHttp's HTTP/1.1 upgrade on chat.signal.org), the
//     server-pushed provisioning address, padded keepalives and a PING;
//   - the enclaves get a credential-less WebSocket upgrade, the other Signal-hosted services
//     one HTTPS GET.
//
// It records the facts (DNS, TCP, TLS, per-step outcomes); conclusions are left to the backend.
//
// After the targets, address controls (controls.go) show which method blocked a target that
// failed before Signal answered: DNS-over-HTTPS and 1.1.1.1 lookups, the clients' static
// addresses (static.go), a decoy SNI, another Signal client's hello, a browser's hello and TLS
// 1.2. On request, the circumvention routes (circumvention.go) measure Signal's domain fronts:
// libsignal's reflector tunnels (tunnel.go) and Signal-Android's fronted requests. Options that
// take lists are parsed in options.go; trust anchors, including SignalCA, are in roots.go.
//
// The previous experiment, signal (ts-029), keeps running unchanged: this one changes what the
// measurement covers, so it is a new experiment with its own spec rather than a new version.
package signalclient
