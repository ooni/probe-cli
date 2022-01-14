package model

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/lucas-clemente/quic-go"
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

// SimpleDialer establishes network connections.
type SimpleDialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer is a SimpleDialer with the possibility of closing open connections.
type Dialer interface {
	// A Dialer is also a SimpleDialer.
	SimpleDialer

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// HTTPClient is an http.Client-like interface.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// HTTPTransport is an http.Transport-like structure.
type HTTPTransport interface {
	// Network returns the network used by the transport, which
	// should be one of "tcp" and "quic".
	Network() string

	// RoundTrip performs the HTTP round trip.
	RoundTrip(req *http.Request) (*http.Response, error)

	// CloseIdleConnections closes idle connections.
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

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPLikeConn.
	Listen(addr *net.UDPAddr) (UDPLikeConn, error)
}

// QUICDialer dials QUIC sessions.
type QUICDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	//
	// Recommended tlsConfig setup:
	//
	// - set ServerName to be the SNI;
	//
	// - set RootCAs to NewDefaultCertPool();
	//
	// - set NextProtos to []string{"h3"}.
	//
	// Typically, you want to pass `&quic.Config{}` as quicConfig.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)

	// Network returns the resolver type (e.g., system, dot, doh).
	Network() string

	// Address returns the resolver address (e.g., 8.8.8.8:53).
	Address() string

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()

	// LookupHTTPS issues an HTTPS query for a domain.
	LookupHTTPS(
		ctx context.Context, domain string) (*HTTPSSvc, error)
}

// TLSDialer is a Dialer dialing TLS connections.
type TLSDialer interface {
	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()

	// DialTLSContext dials a TLS connection. This method will always return
	// to you a oohttp.TLSConn, so you can always safely cast to it.
	DialTLSContext(ctx context.Context, network, address string) (net.Conn, error)
}

// TLSHandshaker is the generic TLS handshaker.
type TLSHandshaker interface {
	// Handshake creates a new TLS connection from the given connection and
	// the given config. This function DOES NOT take ownership of the connection
	// and it's your responsibility to close it on failure.
	//
	// Recommended tlsConfig setup:
	//
	// - set ServerName to be the SNI;
	//
	// - set RootCAs to NewDefaultCertPool();
	//
	// - set NextProtos to []string{"h2", "http/1.1"} for HTTPS
	// and []string{"dot"} for DNS-over-TLS.
	//
	// QUIRK: The returned connection will always implement the TLSConn interface
	// exposed by ooni/oohttp. A future version of this interface may instead
	// return directly a TLSConn to avoid unconditional castings.
	Handshake(ctx context.Context, conn net.Conn, tlsConfig *tls.Config) (
		net.Conn, tls.ConnectionState, error)
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

// UnderlyingNetworkLibrary defines the basic functionality from
// which the network extensions depend. By changing the default
// implementation of this interface, we can implement a wide array
// of tests, including self censorship tests.
type UnderlyingNetworkLibrary interface {
	// ListenUDP creates a new model.UDPLikeConn conn.
	ListenUDP(network string, laddr *net.UDPAddr) (UDPLikeConn, error)

	// LookupHost lookups a domain using the stdlib resolver.
	LookupHost(ctx context.Context, domain string) ([]string, error)

	// NewSimpleDialer returns a new SimpleDialer.
	NewSimpleDialer(timeout time.Duration) SimpleDialer
}
