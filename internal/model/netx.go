package model

import (
	"context"
	"net"
	"syscall"
)

//
// Network extensions
//

// The DNSDecoder decodes DNS replies.
type DNSDecoder interface {
	// DecodeLookupHost decodes an A or AAAA reply.
	//
	// Arguments:
	//
	// - qtype is the query type (e.g., dns.TypeAAAA)
	//
	// - data contains the reply bytes read from a DNSTransport
	//
	// Returns:
	//
	// - on success, a list of IP addrs inside the reply and a nil error
	//
	// - on failure, a nil list and an error.
	//
	// Note that this function will return an error if there is no
	// IP address inside of the reply.
	DecodeLookupHost(qtype uint16, data []byte) ([]string, error)

	// DecodeHTTPS decodes an HTTPS reply.
	//
	// The argument is the reply as read by the DNSTransport.
	//
	// On success, this function returns an HTTPSSvc structure and
	// a nil error. On failure, the HTTPSSvc pointer is nil and
	// the error points to the error that occurred.
	//
	// This function will return an error if the HTTPS reply does not
	// contain at least a valid ALPN entry. It will not return
	// an error, though, when there are no IPv4/IPv6 hints in the reply.
	DecodeHTTPS(data []byte) (*HTTPSSvc, error)
}

// The DNSEncoder encodes DNS queries to bytes
type DNSEncoder interface {
	// Encode transforms its arguments into a serialized DNS query.
	//
	// Arguments:
	//
	// - domain is the domain for the query (e.g., x.org);
	//
	// - qtype is the query type (e.g., dns.TypeA);
	//
	// - padding is whether to add padding to the query.
	//
	// On success, this function returns a valid byte array and
	// a nil error. On failure, we have an error and the byte array is nil.
	Encode(domain string, qtype uint16, padding bool) ([]byte, error)
}

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

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// HTTPSSvc is the reply to an HTTPS DNS query.
type HTTPSSvc struct {
	// ALPN contains the ALPNs inside the HTTPS reply.
	ALPN []string

	// IPv4 contains the IPv4 hints (which may be empty).
	IPv4 []string

	// IPv6 contains the IPv6 hints (which may be empty).
	IPv6 []string
}

// UDPLikeConn is a net.PacketConn with some extra functions
// required to convince the QUIC library (lucas-clemente/quic-go)
// to inflate the receive buffer of the connection.
//
// The QUIC library will treat this connection as a "dumb"
// net.PacketConn, calling its ReadFrom and WriteTo methods
// as opposed to more efficient methods that are available
// under Linux and (maybe?) FreeBSD.
//
// It seems fine to avoid performance optimizations, because
// they would complicate the implementation on our side and
// our use cases (blocking and heavy throttling) do not seem
// to require such optimizations.
//
// See https://github.com/ooni/probe/issues/1754 for a more
// comprehensive discussion of UDPLikeConn.
type UDPLikeConn interface {
	// An UDPLikeConn is a net.PacketConn conn.
	net.PacketConn

	// SetReadBuffer allows setting the read buffer.
	SetReadBuffer(bytes int) error

	// SyscallConn returns a conn suitable for calling syscalls,
	// which is also instrumental to setting the read buffer.
	SyscallConn() (syscall.RawConn, error)
}
