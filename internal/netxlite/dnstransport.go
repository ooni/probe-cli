package netxlite

import "context"

// DNSTransport represents an abstract DNS transport.
type DNSTransport interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(ctx context.Context, query []byte) (reply []byte, err error)

	// RequiresPadding return true for DoH and DoT according to RFC8467
	RequiresPadding() bool

	// Network is the network of the round tripper (e.g. "dot")
	Network() string

	// Address is the address of the round tripper (e.g. "1.1.1.1:853")
	Address() string

	// CloseIdleConnections closes idle connections.
	CloseIdleConnections()
}
