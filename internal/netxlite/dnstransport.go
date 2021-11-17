package netxlite

import "context"

// DNSTransport represents an abstract DNS transport.
type DNSTransport interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(ctx context.Context, query []byte) (reply []byte, err error)

	// RequiresPadding returns whether this transport needs padding.
	RequiresPadding() bool

	// Network is the network of the round tripper (e.g. "dot").
	Network() string

	// Address is the address of the round tripper (e.g. "1.1.1.1:853").
	Address() string

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}
