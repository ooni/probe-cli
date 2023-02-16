package model

//
// Network extensions
//

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"syscall"
	"time"

	"github.com/lucas-clemente/quic-go"
)

// DNSResponse is a parsed DNS response ready for further processing.
type DNSResponse interface {
	// Query is the query associated with this response.
	Query() DNSQuery

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

	// DecodeCNAME returns the first CNAME entry in this response.
	DecodeCNAME() (string, error)
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
	// The value returned by this function WILL be memoized after the first call,
	// so you SHOULD create a new DNSQuery if you need to retry a query.
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
	// You serialize to bytes using DNSQuery.Bytes. This operation MAY fail
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

// DNSTransportWrapper is a type that takes in input a DNSTransport
// and returns in output a wrapped DNSTransport.
type DNSTransportWrapper interface {
	WrapDNSTransport(txp DNSTransport) DNSTransport
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

// DialerWrapper is a type that takes in input a Dialer
// and returns in output a wrapped Dialer.
type DialerWrapper interface {
	WrapDialer(d Dialer) Dialer
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
	// should be one of "tcp" and "udp".
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

// QUICDialerWrapper is a type that takes in input a QUICDialer
// and returns in output a wrapped QUICDialer.
type QUICDialerWrapper interface {
	WrapQUICDialer(qd QUICDialer) QUICDialer
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
	// - set RootCAs to nil (which causes us to use the default cert pool);
	//
	// - set NextProtos to []string{"h3"}.
	//
	// Typically, you want to pass `&quic.Config{}` as quicConfig.
	DialContext(ctx context.Context, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error)

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()
}

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)

	// Network returns the resolver type. It should be one of:
	//
	// - go: means we're using whatever resolver the Go stdlib uses
	// depending on the current build configuration;
	//
	// - system: means we've been compiled with `CGO_ENABLED=1`
	// so we can bypass the go resolver and call getaddrinfo directly;
	//
	// - udp: is a custom DNS-over-UDP resolver;
	//
	// - tcp: is a custom DNS-over-TCP resolver;
	//
	// - dot: is a custom DNS-over-TLS resolver;
	//
	// - doh: is a custom DNS-over-HTTPS resolver;
	//
	// - doh3: is a custom DNS-over-HTTP3 resolver.
	//
	// See https://github.com/ooni/probe/issues/2029#issuecomment-1140805266
	// for an explanation of why it would not be proper to call "netgo" the
	// resolver we get by default from the standard library.
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
	// - set RootCAs to nil (which causes us to use the default cert pool);
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

// Trace allows to collect measurement traces. A trace is injected into
// netx operations using context.WithValue. Netx code retrieves the trace
// using context.Value. See docs/design/dd-003-step-by-step.md for the
// design document explaining why we implemented context-based tracing.
type Trace interface {
	// TimeNow returns the current time. Normally, this should be the same
	// value returned by time.Now but you may want to manipulate the time
	// returned when testing to have deterministic tests. To this end, you
	// can use functionality exported by the ./internal/testingx pkg.
	TimeNow() time.Time

	// MaybeWrapNetConn possibly wraps a net.Conn with the caller trace. If there's no
	// desire to wrap the net.Conn, this function just returns the original net.Conn.
	//
	// Arguments:
	//
	// - conn is the non-nil underlying net.Conn to be wrapped
	MaybeWrapNetConn(conn net.Conn) net.Conn

	// MaybeWrapUDPLikeConn is like MaybeWrapNetConn but for UDPLikeConn.
	//
	// Arguments:
	//
	// - conn is the non-nil underlying UDPLikeConn to be wrapped
	MaybeWrapUDPLikeConn(conn UDPLikeConn) UDPLikeConn

	// OnDNSRoundTripForLookupHost is used with a DNSTransport and called
	// when the RoundTrip terminates.
	//
	// Arguments:
	//
	// - started is when we called transport.RoundTrip
	//
	// - reso is the parent resolver for the trace;
	//
	// - query is the non-nil DNS query we use for the RoundTrip
	//
	// - response is a valid DNS response, obtained after the RoundTrip;
	//
	// - addrs is the list of addresses obtained after the RoundTrip, which
	// is empty if the RoundTrip failed
	//
	// - err is the result of DNSLookup; either an error or nil
	//
	// - finished is the time right after the RoundTrip
	OnDNSRoundTripForLookupHost(started time.Time, reso Resolver, query DNSQuery,
		response DNSResponse, addrs []string, err error, finished time.Time)

	// OnDelayedDNSResponse is used with a DNSOverUDPTransport and called
	// when we get delayed, unexpected DNS responses.
	//
	// Arguments:
	//
	// - started is when we started reading the delayed response;
	//
	// - txp is the DNS transport used with the resolver;
	//
	// - query is the non-nil DNS query we use for the RoundTrip;
	//
	// - response is the non-nil valid DNS response, obtained after some delay;
	//
	// - addrs is the list of addresses obtained after decoding the delayed response,
	// which is empty if the response did not contain any addresses, which we
	// extract by calling the DecodeLookupHost method.
	//
	// - err is the result of DecodeLookupHost: either an error or nil;
	//
	// - finished is when we have read the delayed response.
	OnDelayedDNSResponse(started time.Time, txp DNSTransport, query DNSQuery,
		resp DNSResponse, addrs []string, err error, finsihed time.Time) error

	// OnConnectDone is called when connect terminates.
	//
	// Arguments:
	//
	// - started is when we called connect;
	//
	// - network is the network we're using (one of "tcp" and "udp");
	//
	// - domain is the domain for which we're calling connect. If the user called
	// connect for an IP address and a port, then domain will be an IP address;
	//
	// - remoteAddr is the TCP endpoint with which we are connecting: it will
	// consist of an IP address and a port (e.g., 8.8.8.8:443, [::1]:5421);
	//
	// - err is the result of connect: either an error or nil;
	//
	// - finished is when connect returned.
	//
	// The error passed to this function will always be wrapped such that the
	// string returned by Error is an OONI error.
	OnConnectDone(
		started time.Time, network, domain, remoteAddr string, err error, finished time.Time)

	// OnTLSHandshakeStart is called when the TLS handshake starts.
	//
	// Arguments:
	//
	// - now is the moment before we start the handshake;
	//
	// - remoteAddr is the TCP endpoint with which we are connecting: it will
	// consist of an IP address and a port (e.g., 8.8.8.8:443, [::1]:5421);
	//
	// - config is the non-nil TLS config we're using.
	OnTLSHandshakeStart(now time.Time, remoteAddr string, config *tls.Config)

	// OnTLSHandshakeDone is called when the TLS handshake terminates.
	//
	// Arguments:
	//
	// - started is when we started the handshake;
	//
	// - remoteAddr is the TCP endpoint with which we are connecting: it will
	// consist of an IP address and a port (e.g., 8.8.8.8:443, [::1]:5421);
	//
	// - config is the non-nil TLS config we're using;
	//
	// - state is the state of the TLS connection after the handshake, where all
	// fields are zero-initialized if the handshake failed;
	//
	// - err is the result of the handshake: either an error or nil;
	//
	// - finished is right after the handshake.
	//
	// The error passed to this function will always be wrapped such that the
	// string returned by Error is an OONI error.
	OnTLSHandshakeDone(started time.Time, remoteAddr string, config *tls.Config,
		state tls.ConnectionState, err error, finished time.Time)

	// OnQUICHandshakeStart is called before the QUIC handshake.
	//
	// Arguments:
	//
	// - now is the moment before we start the handshake;
	//
	// - remoteAddr is the QUIC endpoint with which we are connecting: it will
	// consist of an IP address and a port (e.g., 8.8.8.8:443, [::1]:5421);
	//
	// - config is the possibly-nil QUIC config we're using.
	OnQUICHandshakeStart(now time.Time, remoteAddr string, quicConfig *quic.Config)

	// OnQUICHandshakeDone is called after the QUIC handshake.
	//
	// Arguments:
	//
	// - started is when we started the handshake;
	//
	// - remoteAddr is the QUIC endpoint with which we are connecting: it will
	// consist of an IP address and a port (e.g., 8.8.8.8:443, [::1]:5421);
	//
	// - qconn is the QUIC connection we receive after the handshake: either
	// a valid quic.EarlyConnection or nil;
	//
	// - config is the non-nil TLS config we are using;
	//
	// - err is the result of the handshake: either an error or nil;
	//
	// - finished is right after the handshake.
	//
	// The error passed to this function will always be wrapped such that the
	// string returned by Error is an OONI error.
	OnQUICHandshakeDone(started time.Time, remoteAddr string, qconn quic.EarlyConnection,
		config *tls.Config, err error, finished time.Time)
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

// UnderlyingNetwork implements the underlying network APIs on
// top of which we implement network extensions.
type UnderlyingNetwork interface {
	// DefaultCertPool returns the underlying cert pool used by the
	// network extensions library. You MUST NOT use this function to
	// modify the default cert pool since this would lead to a data
	// race. Use [netxlite.NewDefaultCertPool] if you wish to get
	// a copy of the default cert pool that you can modify.
	DefaultCertPool() *x509.CertPool

	// DialContext is equivalent to net.Dialer.DialContext except that
	// there is also an explicit timeout for dialing.
	DialContext(ctx context.Context, timeout time.Duration, network, address string) (net.Conn, error)

	// GetaddrinfoLookupANY is like net.Resolver.LookupHost except that it
	// also returns to the caller the CNAME when it is available.
	GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error)

	// GetaddrinfoResolverNetwork returns the resolver network.
	GetaddrinfoResolverNetwork() string

	// ListenUDP is equivalent to net.ListenUDP.
	ListenUDP(network string, addr *net.UDPAddr) (UDPLikeConn, error)
}
