package model

//
// Network extensions
//

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/miekg/dns"
)

// DNSResponse is a parsed DNS response ready for further processing.
type DNSResponse interface {
	// Query is the query associated with this response.
	Query() DNSQuery

	// Message returns the underlying DNS message.
	Message() *dns.Msg

	// Bytes returns the bytes from which we parsed the query.
	Bytes() []byte

	// Rcode returns the response's Rcode.
	Rcode() int

	// DecodeHTTPS returns information gathered from all the HTTPS
	// records found inside of this response.
	DecodeHTTPS() (*HTTPSSvc, error)

	// DecodeLookupHost returns the addresses in the response matching
	// the original query type (one of A and AAAA).
	DecodeLookupHost() ([]string, error)

	// DecodeNS returns all the NS entries in this response.
	DecodeNS() ([]*net.NS, error)
}

// The DNSDecoder decodes DNS responses.
type DNSDecoder interface {
	// DecodeResponse decodes a DNS response message.
	//
	// Arguments:
	//
	// - data is the raw reply
	//
	// This function fails if we cannot parse data as a DNS
	// message or the message is not a response.
	//
	// Regarding the returned response, remember that the Rcode
	// MAY still be nonzero (this method does not treat a nonzero
	// Rcode as an error when parsing the response).
	DecodeResponse(data []byte, query DNSQuery) (DNSResponse, error)
}

// DNSQuery is an encoded DNS query ready to be sent using a DNSTransport.
type DNSQuery interface {
	// Domain is the domain we're querying for.
	Domain() string

	// Type is the query type.
	Type() uint16

	// Bytes serializes the query to bytes. This function may fail if we're not
	// able to correctly encode the domain into a query message.
	//
	// The value returned by this function MAY be memoized after the first call.
	//
	// Note: every time you serialize this query you ALWAYS obtain the same
	// query ID and, in general, you SHOULD NOT reuse IDs.
	Bytes() ([]byte, error)

	// ID returns the query ID.
	ID() uint16
}

// The DNSEncoder encodes DNS queries to bytes
type DNSEncoder interface {
	// Encode transforms its arguments into a serialized DNS query.
	//
	// Every time you call Encode, you get a new DNSQuery value
	// using a query ID selected at random.
	//
	// Serialization to bytes is lazy to acommodate DNS transports that
	// do not need to serialize and send bytes, e.g., getaddrinfo.
	//
	// You serialized to bytes using DNSQuery.Bytes. This operation MAY fail
	// if the domain name cannot be packed into a DNS message (e.g., it is
	// too long to fit into the message).
	//
	// Arguments:
	//
	// - domain is the domain for the query (e.g., x.org);
	//
	// - qtype is the query type (e.g., dns.TypeA);
	//
	// - padding is whether to add padding to the query.
	//
	// This function will transform the domain into an FQDN is it's not
	// already expressed in the FQDN format.
	Encode(domain string, qtype uint16, padding bool) DNSQuery
}

// DNSTransport represents an abstract DNS transport.
type DNSTransport interface {
	// RoundTrip sends a DNS query and receives the reply.
	RoundTrip(ctx context.Context, query DNSQuery) (DNSResponse, error)

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
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error)

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

	// LookupNS issues a NS query for a domain.
	LookupNS(ctx context.Context, domain string) ([]*net.NS, error)
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
