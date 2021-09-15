package measure

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// Measurer performs measurements. Make sure you fill all the
// fields labelled as MANDATORY before using a Measurer.
//
// The typical usage is that you create a Measurer and run a
// single measurement with it. If you settle for more complex
// usage patterns, you need to take care of not having
// overlapping traces. So, the suggested usage is the simplest.
//
// CAVEAT: the Measurer is not designed to be used by multiple
// goroutines at the same time. A future version of this codebase
// MAY provide more guarantees in this regard.
type Measurer struct {
	// Begin is the MANDATORY time when we started measuring, which
	// is used as the base for computing the elapsed time.
	Begin time.Time

	// Logger is the MANDATORY logger to use.
	Logger Logger

	// Connector is the MANDATORY Connector to use. You can create
	// a connector using the NewConnector factory function.
	Connector Connector

	// TLSHandshaker is the MANDATORY TLSHandshaker to use. You can
	// create an handshaker using NewTLSHandshakerStdlib.
	TLSHandshaker TLSHandshaker

	// QUICHandshaker is the MANDATORY QUICHandshaker to use. You can
	// create an handshaker using NewQUICHandshaker.
	QUICHandshaker QUICHandshaker

	// Trace is the MANDATORY Trace to use. You can create a
	// new trace instance using the NewTrace factory function.
	Trace *Trace
}

// NewMeasurerStdlib creates a new Measurer instance configured
// to use the standard library for managing TLS connections.
//
// The begin param is the beginning of time. We use this info
// to compute the elapsed time of events we observe.
//
// The logger param prints logs.
//
// Do not pass to this factory nil or empty parameters.
func NewMeasurerStdlib(begin time.Time, logger Logger) *Measurer {
	trace := NewTrace(begin)
	return &Measurer{
		Begin:          begin,
		Logger:         logger,
		Connector:      NewConnector(begin, logger, trace),
		TLSHandshaker:  NewTLSHandshakerStdlib(begin, logger),
		QUICHandshaker: NewQUICHandshaker(begin, logger, trace),
		Trace:          trace,
	}
}

// MergeEndpoints takes in input the result of multiple DNS resolutions
// and a port number and creates a list of unique endpoints.
func MergeEndpoints(all []*LookupHostResult, port string) (epnts []string) {
	freq := make(map[string]int)
	for _, one := range all {
		for _, addr := range one.Addrs {
			endpoint := net.JoinHostPort(addr, port)
			freq[endpoint]++
		}
	}
	for epnt := range freq {
		epnts = append(epnts, epnt)
	}
	return
}

// topLevelResult is the interface that all top-level
// results must expose to simplify their usage.
type topLevelResult interface {
	Successful() bool
}

// ParseURLResult is the result of ParseURL.
type ParseURLResult struct {
	// URL is the original URL.
	URL string `json:"url"`

	// Failure is the error that occurred.
	Failure error `json:"failure"`

	// Parsed is the parsed URL.
	Parsed *url.URL `json:"-"`

	// Port is the port that was present inside
	// the URL or that we inferred from the scheme.
	Port string `json:"port"`
}

var _ topLevelResult = &ParseURLResult{}

// Successful returns true when no error occurred.
func (m *ParseURLResult) Successful() bool {
	return m.Failure == nil
}

// Hostname returns the underlying URL's hostname if the
// parsing was successful and panics otherwise.
func (m *ParseURLResult) Hostname() string {
	return m.Parsed.Hostname()
}

// ParseURL parses the URL and returns the results. This function
// will, in particular, figure out which port should be used for
// the related endpoint. If it cannot figure out the port, the code
// will fail and the Failure field will be set accordingly.
func ParseURL(URL string) *ParseURLResult {
	m := &ParseURLResult{URL: URL}
	m.Parsed, m.Failure = url.Parse(URL)
	if m.Failure != nil {
		return m
	}
	m.Port = m.Parsed.Port()
	if m.Port == "" {
		switch m.Parsed.Scheme {
		case "http":
			m.Port = "80"
		case "https":
			m.Port = "443"
		default:
			// TODO: add proper factory
			m.Failure = &errorsx.ErrWrapper{
				Failure:    "url_missing_port_error", // TODO: add
				Operation:  errorsx.TopLevelOperation,
				WrappedErr: errors.New("cannot guess port"),
			}
		}
	}
	return m
}

// LookupHostResult is the result of Measurer.LookupHostSystem,
// Measurer.LookupHostUDP and other similar functions.
type LookupHostResult struct {
	// Engine is the engine we have used. This is set to
	// "system", "udp", "tcp", "dot", or "doh".
	Engine string `json:"engine"`

	// Address is the address of the remote DNS server. It is
	// empty when we're using the system resolver.
	Address string `json:"address,omitempty"`

	// QueryTypeInt is the query type as int. This field is
	// not exported to JSON because we export the string.
	QueryTypeInt uint16 `json:"-"`

	// QueryTypeString is the query type as string. This field
	// is mainly used for exporting an understandable JSON.
	QueryTypeString string `json:"query_type,omitempty"`

	// Domain is the domain to resolve.
	Domain string `json:"domain"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Query contains the raw query. This field is nil when
	// we are using the system resolver.
	Query []byte `json:"query,omitempty"`

	// Failure contains the error that occurred. This
	// field is nil if there was no error.
	Failure error `json:"failure"`

	// Addrs contains the resolved addresses. This field is
	// nil when we the resolve operation failed.
	Addrs []string `json:"addrs"`

	// Reply contains the raw reply. This field is nil when
	// we are using the system resolver.
	Reply []byte `json:"reply,omitempty"`

	// UDPConnect contains UDP connect events. This field is nil when
	// we are not using a resolver that uses UDP.
	UDPConnect *ConnectResult `json:"udp_connect,omitempty"`

	// NetworkEvents contains I/O events. This field is nil when
	// we are using the system resolver.
	NetworkEvents []*TraceEntry `json:"network_events,omitempty"`
}

var _ topLevelResult = &LookupHostResult{}

// Successful returns true when no error occurred.
func (m *LookupHostResult) Successful() bool {
	return m.Failure == nil
}

// LookupHostSystem measures the effects of resolving the given
// host name using the system resolver (i.e., getaddrinfo).
func (mx *Measurer) LookupHostSystem(ctx context.Context, host string) *LookupHostResult {
	r := &dnsxResolverSystem{begin: mx.Begin, logger: mx.Logger}
	return r.LookupHost(ctx, host)
}

// LookupHostUDP measures the effect of sending a query to the
// given resolverAddr (e.g., "1.1.1.1:443") for the given query
// type `qtype` (e.g., dns.TypeA) and the given host name.
func (mx *Measurer) LookupHostUDP(ctx context.Context,
	host string, qtype uint16, resolverAddr string) *LookupHostResult {
	udpConnect := mx.Connector.Connect(ctx, "udp", resolverAddr)
	if udpConnect.Failure != nil {
		return &LookupHostResult{
			Failure:    udpConnect.Failure,
			UDPConnect: udpConnect,
		}
	}
	defer udpConnect.Conn.Close()
	dnsx := newDNSXTransportWithUDPConn(mx.Begin, udpConnect.Conn)
	m := dnsx.LookupHost(ctx, host, qtype)
	m.NetworkEvents = append(m.NetworkEvents, mx.Trace.ExtractEvents()...)
	return m
}

// TCPConnectResult is the result of a Measurer.TCPConnect operation.
type TCPConnectResult struct {
	*ConnectResult
}

// Successful returns true when no error occurred.
func (m *TCPConnectResult) Successful() bool {
	return m.Failure == nil
}

var _ topLevelResult = &TCPConnectResult{}

// TCPConnect measures the effects of creating a TCP connection with
// the given address (e.g., "1.1.1.1:443").
//
// This function closes the underlying TCP conn when done
// so no cleanup action is required once it returns.
func (mx *Measurer) TCPConnect(ctx context.Context, address string) *TCPConnectResult {
	mtcp := mx.Connector.Connect(ctx, "tcp", address)
	m := &TCPConnectResult{mtcp}
	if m.Conn != nil {
		m.Conn.Close()
	}
	return m
}

// TLSEndpointDialResult is the result of a Measurer.TLSEndpointDial operation.
type TLSEndpointDialResult struct {
	// TCPConnect contains the result of the TCP connect operation.
	TCPConnect *ConnectResult `json:"tcp_connect"`

	// TLSHandshake contains the result of the TLS handshake operation.
	TLSHandshake *TLSHandshakeResult `json:"tls_handshake"`

	// NetworkEvents contains network I/O events.
	NetworkEvents []*TraceEntry `json:"network_events"`
}

var _ topLevelResult = &TLSEndpointDialResult{}

// Successful returns true when no error occurred.
func (m *TLSEndpointDialResult) Successful() bool {
	return m.TCPConnect != nil && m.TCPConnect.Failure == nil &&
		m.TLSHandshake != nil && m.TLSHandshake.Failure == nil
}

// TLSEndpointDial measures the effects of creating a TCP connection with
// the given address (e.g., "1.1.1.1:443") and then performing a TLS
// handshake with it using the given TLS config. You SHOULD set
// the following config fields:
//
// - ServerName to the desired SNI or InsecureSkipVerify to
// skip the certificate name verification;
//
// - RootCAs to nextlite.NewDefaultCertPool() output;
//
// - NextProtos to the desired ALPN ([]string{"h2", "http/1.1"} for
// HTTPS and []string{"dot"} for DNS-over-TLS).
//
// However, note that if mx.TLSHandshaker is not the output
// of NewTLSHandshakerStdlib, we will be doing parroting, hence
// some ClientHello fields may differ from the one you
// configured for obvious parroting reasons.
//
// This function closes the underlying TLS conn when done
// so no cleanup action is required once it returns.
func (mx *Measurer) TLSEndpointDial(ctx context.Context,
	address string, config *tls.Config) *TLSEndpointDialResult {
	m := mx.tlsEndpointDial(ctx, address, config)
	if m.TLSHandshake != nil && m.TLSHandshake.Conn != nil {
		m.TLSHandshake.Conn.Close()
	}
	return m
}

// tlsEndpointDial is a TLSEndpointDial that does not close the underlying conn.
func (mx *Measurer) tlsEndpointDial(ctx context.Context,
	address string, config *tls.Config) *TLSEndpointDialResult {
	m := &TLSEndpointDialResult{}
	m.TCPConnect = mx.Connector.Connect(ctx, "tcp", address)
	if m.TCPConnect.Failure != nil {
		return m
	}
	m.TLSHandshake = mx.TLSHandshaker.TLSHandshake(
		ctx, m.TCPConnect.Conn, config)
	m.NetworkEvents = append(m.NetworkEvents, mx.Trace.ExtractEvents()...)
	if m.TLSHandshake.Failure != nil {
		m.TCPConnect.Conn.Close()
		// fallthrough
	}
	return m
}

// QUICEndpointDialResult is the result of Measurer.QUICEndpointDial.
type QUICEndpointDialResult struct {
	// QUICHandshake contains the result of the QUIC handshake operation.
	QUICHandshake *QUICHandshakeResult `json:"quic_handshake"`

	// NetworkEvents contains network I/O events.
	NetworkEvents []*TraceEntry `json:"network_events"`
}

var _ topLevelResult = &LookupHostResult{}

// Successful returns true when no error occurred.
func (m *QUICEndpointDialResult) Successful() bool {
	return m.QUICHandshake != nil && m.QUICHandshake.Failure == nil
}

// QUICEndpointDial measures the effects of creating a QUIC session with
// the given address (e.g., "1.1.1.1:443") and config. You SHOULD set
//
// - ServerName to the desired SNI or InsecureSkipVerify to
// skip the certificate name verification;
//
// - RootCAs to nextlite.NewDefaultCertPool() output;
//
// - NextProtos to []string{"h3"} set the desired ALPN.
//
// This function closes the underlying QUIC session when done
// so no cleanup action is required once it returns.
func (mx *Measurer) QUICEndpointDial(ctx context.Context,
	address string, config *tls.Config) *QUICEndpointDialResult {
	m := mx.quicEndpointDial(ctx, address, config)
	if m.QUICHandshake.Sess != nil {
		m.QUICHandshake.Sess.CloseWithError(0, "")
	}
	return m
}

// quicEndpointDial is a QUICEndpointDial that does not close the QUIC session.
func (mx *Measurer) quicEndpointDial(ctx context.Context,
	address string, config *tls.Config) *QUICEndpointDialResult {
	m := &QUICEndpointDialResult{}
	m.QUICHandshake = mx.QUICHandshaker.QUICHandshake(ctx, address, config)
	m.NetworkEvents = append(m.NetworkEvents, mx.Trace.ExtractEvents()...)
	return m
}

// HTTPEndpointGetResult is the result of Measurer.HTTPEndpointGet,
// Measurer.HTTPSEndpointGet, and Measurer.HTTP3EndpointGet.
type HTTPEndpointGetResult struct {
	// TCPConnect contains the result of the TCP connect operation. This
	// field is not present when we are using HTTP3.
	TCPConnect *ConnectResult `json:"tcp_connect,omitempty"`

	// TLSHandshake contains the result of the TLS handshake operation. This
	// field is not present when we are using HTTP or HTTP3.
	TLSHandshake *TLSHandshakeResult `json:"tls_handshake,omitempty"`

	// QUICHandshake contains the result of the QUIC handshake operation. This
	// field is not present when we are using HTTP or HTTPS.
	QUICHandshake *QUICHandshakeResult `json:"quic_handshake,omitempty"`

	// NetworkEvents contains network I/O events.
	NetworkEvents []*TraceEntry `json:"network_events"`

	// HTTP contains the HTTP request and the HTTP response.
	HTTP *HTTPRequestResponse `json:"http"`
}

// Successful returns true when no error occurred.
func (m *HTTPEndpointGetResult) Successful() bool {
	return m.HTTP != nil && m.HTTP.Failure == nil // only check final stage
}

// HTTPSEndpointGet measures the effects of connecting to the given
// address, performing a TLS handshake with it, and then sending the
// given request and receiving the corresponding response.
//
// See TLSEndpointDial docs for information about the expected content
// of the address and config parameters.
//
// See HTTPRequest docs for information about the mandatory fields.
func (mx *Measurer) HTTPSEndpointGet(ctx context.Context, address string,
	config *tls.Config, request *HTTPRequest) *HTTPEndpointGetResult {
	tlsm := mx.tlsEndpointDial(ctx, address, config)
	m := &HTTPEndpointGetResult{
		TCPConnect:    tlsm.TCPConnect,
		TLSHandshake:  tlsm.TLSHandshake,
		QUICHandshake: nil,
		NetworkEvents: tlsm.NetworkEvents,
	}
	if !tlsm.Successful() {
		return m
	}
	defer m.TLSHandshake.Conn.Close()
	txp := newHTTPTransportWithTLSConn(mx.Logger, m.TLSHandshake.Conn)
	clnt := newHTTPClient(mx.Begin, txp)
	m.HTTP = clnt.Get(ctx, request)
	m.NetworkEvents = append(m.NetworkEvents, mx.Trace.ExtractEvents()...)
	return m
}
