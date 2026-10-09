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
// The previous experiment, signal (ts-029), keeps running unchanged: this one changes what the
// measurement covers, so it is a new experiment with its own spec rather than a new version.
package signalclient
