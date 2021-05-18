package netplumbing

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	// TraceKindConnect identifies a trace collected during connect.
	TraceKindConnect = "connect"

	// TraceKindHTTPRoundTrip is a trace collected during the HTTP round trip.
	TraceKindHTTPRoundTrip = "http_round_trip"

	// TraceKindRead identifies a trace collected during read.
	TraceKindRead = "read"

	// TraceKindResolve identifies a trace collected during resolve.
	TraceKindResolve = "resolve"

	// TraceKindTLSHandshake identifies a trace collected during a TLS handshake.
	TraceKindTLSHandshake = "tls_handshake"

	// TraceKindWrite identifies a trace collected during write.
	TraceKindWrite = "write"
)

// TraceEvent is an event occurred when tracing.
type TraceEvent interface {
	// Kind returns the event kind.
	Kind() string
}

// Tracer traces network events.
type Tracer struct {
	// Connector is the mandatory underlying connector.
	Connector Connector

	// HTTPTransport is the mandatory underlying http.Transport.
	HTTPTransport http.RoundTripper

	// MaxBodySize contains the maximum body size for response
	// bodies. If not set, we use a reasonable default.
	MaxBodySize int

	// Resolver is the mandatory underlying resolver.
	Resolver Resolver

	// TLSHandshaker is the mandatory underlying TLS handshaker.
	TLSHandshaker TLSHandshaker

	// events contains the events collected so far.
	events []TraceEvent

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// add adds an event to the trace.
func (tr *Tracer) add(ev TraceEvent) {
	defer tr.mu.Unlock()
	tr.mu.Lock()
	tr.events = append(tr.events, ev)
}

// MoveOut moves the collected events out of the trace.
func (tr *Tracer) MoveOut() []TraceEvent {
	defer tr.mu.Unlock()
	tr.mu.Lock()
	out := tr.events
	tr.events = nil
	return out
}

// NewTracer creates an new instace of tracer that is using
// as its Connector, HTTPTransport, Resolver, and TLSHandshaker
// the defaults using by the current transport.
func (txp *Transport) NewTracer() *Tracer {
	return &Tracer{
		Connector:     txp.DefaultConnector(),
		HTTPTransport: txp.RoundTripper,
		Resolver:      txp.DefaultResolver(),
		TLSHandshaker: txp.DefaultTLSHandshaker(),
	}
}

// NewConfig contains a new Config instance using this Tracer
// as its Connector, HTTPTransport, Resolver, and TLSHandshaker.
func (tr *Tracer) NewConfig() *Config {
	return &Config{
		Connector:     tr,
		HTTPTransport: tr,
		Resolver:      tr,
		TLSHandshaker: tr,
	}
}

// ConnectTrace is a measurement performed during connect.
type ConnectTrace struct {
	// Network is the network we're using (e.g., "tcp")
	Network string

	// DestAddr is the address we're connecting to.
	DestAddr string

	// StartTime is when we started connecting.
	StartTime time.Time

	// EndTime is when we're done.
	EndTime time.Time

	// SourceAddr is the source address when we're connected.
	SourceAddr string

	// Error is the error that occurred.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *ConnectTrace) Kind() string {
	return TraceKindConnect
}

// DialContext implements Connector.DialContext.
func (tr *Tracer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	ev := &ConnectTrace{
		Network:   network,
		DestAddr:  address,
		StartTime: time.Now(),
	}
	defer tr.add(ev)
	conn, err := tr.Connector.DialContext(ctx, network, address)
	ev.EndTime = time.Now()
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.SourceAddr = conn.LocalAddr().String()
	return &tracerConn{Conn: conn, tracer: tr}, nil
}

// tracerConn is a net.Conn that performs tracing.
type tracerConn struct {
	net.Conn
	tracer *Tracer
}

// ReadWriteTrace is a trace collected when reading or writing.
type ReadWriteTrace struct {
	// kind is the structure kind.
	kind string

	// SourceAddr is the source address.
	SourceAddr string

	// DestAddr is the destination address.
	DestAddr string

	// BufferSize is the size of the buffer to send or recv.
	BufferSize int

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done. The duration of the round trip
	// also includes the time spent reading the response.
	EndTime time.Time

	// Count is the number of bytes read or written.
	Count int

	// Error is the error that occurred.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *ReadWriteTrace) Kind() string {
	return te.kind
}

// Read implements net.Conn.Read.
func (c *tracerConn) Read(b []byte) (int, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindRead,
		SourceAddr: c.Conn.LocalAddr().String(),
		DestAddr:   c.Conn.RemoteAddr().String(),
		BufferSize: len(b),
		StartTime:  time.Now(),
	}
	defer c.tracer.add(ev)
	count, err := c.Conn.Read(b)
	ev.EndTime = time.Now()
	ev.Count = count
	ev.Error = err
	return count, err
}

// Write implements net.Conn.Write.
func (c *tracerConn) Write(b []byte) (int, error) {
	ev := &ReadWriteTrace{
		kind:       TraceKindWrite,
		SourceAddr: c.Conn.LocalAddr().String(),
		DestAddr:   c.Conn.RemoteAddr().String(),
		BufferSize: len(b),
		StartTime:  time.Now(),
	}
	defer c.tracer.add(ev)
	count, err := c.Conn.Write(b)
	ev.EndTime = time.Now()
	ev.Count = count
	ev.Error = err
	return count, err
}

// HTTPRoundTripTrace is a measurement collected during the HTTP round trip.
type HTTPRoundTripTrace struct {
	// Method is the request method.
	Method string

	// URL is the request URL.
	URL string

	// RequestHeaders contains the request headers.
	RequestHeaders http.Header

	// RequestBody contains the request body. This body is never
	// truncated but you may wanna truncate it before uploading
	// the measurement to the OONI servers.
	RequestBody []byte

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done. The duration of the round trip
	// also includes the time spent reading the response.
	EndTime time.Time

	// StatusCode contains the status code.
	StatusCode int

	// ResponseHeaders contains the response headers.
	ResponseHeaders http.Header

	// ResponseBody contains the response body. This body is
	// truncated if larger than Tracer.MaxBodySize. You likely
	// want to further truncate it before uploading data to
	// the OONI servers to save bandwidth.
	ResponseBody []byte

	// Error contains the error.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *HTTPRoundTripTrace) Kind() string {
	return TraceKindHTTPRoundTrip
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (tr *Tracer) RoundTrip(req *http.Request) (*http.Response, error) {
	ev := &HTTPRoundTripTrace{
		Method:         req.Method,
		URL:            req.URL.String(),
		RequestHeaders: req.Header,
	}
	defer func() {
		ev.EndTime = time.Now()
		tr.add(ev)
	}()
	if req.Body != nil {
		data, err := ioutil.ReadAll(req.Body)
		if err != nil {
			ev.Error = err
			return nil, err
		}
		ev.RequestBody = data
		req.Body = io.NopCloser(bytes.NewReader(data))
	}
	ev.StartTime = time.Now()
	resp, err := tr.HTTPTransport.RoundTrip(req)
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.StatusCode = resp.StatusCode
	ev.RequestHeaders = resp.Header
	iocloser := resp.Body
	defer iocloser.Close() // close original body
	reader := io.LimitReader(resp.Body, int64(tr.maxBodySize()))
	data, err := ioutil.ReadAll(reader)
	if errors.Is(err, io.EOF) && resp.Close {
		err = nil // we expected to hit the EOF
	}
	if err != nil {
		ev.Error = err
		return nil, err
	}
	ev.ResponseBody = data
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return resp, nil
}

// maxBodySize returns the tr.MaxBodySize or the default max body size.
func (tr *Tracer) maxBodySize() int {
	if tr.MaxBodySize > 0 {
		return tr.MaxBodySize
	}
	return 1 << 24
}

// ResolveTrace is a measurement performed during a DNS resolution.
type ResolveTrace struct {
	// Domain is the domain to resolve.
	Domain string

	// StartTime is when we started the resolve.
	StartTime time.Time

	// EndTime is when we're done.
	EndTime time.Time

	// Addresses contains the resolver addresses.
	Addresses []string

	// Error contains the error.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *ResolveTrace) Kind() string {
	return TraceKindResolve
}

// LookupHost implements Resolver.LookupHost.
func (tr *Tracer) LookupHost(ctx context.Context, domain string) ([]string, error) {
	ev := &ResolveTrace{
		Domain:    domain,
		StartTime: time.Now(),
	}
	defer tr.add(ev)
	addrs, err := tr.Resolver.LookupHost(ctx, domain)
	ev.EndTime = time.Now()
	ev.Addresses = addrs
	ev.Error = err
	return addrs, err
}

// TLSHandshakeTrace is a measurement performed during a TLS handshake.
type TLSHandshakeTrace struct {
	// SourceAddr is the source address.
	SourceAddr string

	// DestAddr is the destination address.
	DestAddr string

	// SkipTLSVerify indicates whether we disabled TLS verification.
	SkipTLSVerify bool

	// ServerName contains the configured server name.
	ServerName string

	// NextProtos contains the protocols for ALPN.
	NextProtos []string

	// StartTime is when we started the TLS handshake.
	StartTime time.Time

	// EndTime is when we're done.
	EndTime time.Time

	// Version contains the TLS version.
	Version uint16

	// CipherSuite contains the negotiated cipher suite.
	CipherSuite uint16

	// NegotiatedProto contains the negotiated proto.
	NegotiatedProto string

	// PeerCerts contains the peer certificates.
	PeerCerts [][]byte

	// Error contains the error.
	Error error
}

// Kind implements TraceEvent.Kind.
func (te *TLSHandshakeTrace) Kind() string {
	return TraceKindTLSHandshake
}

// TLSHanshake implements TLSHandshaker.TLSHandshake.
func (tr *Tracer) TLSHandshake(
	ctx context.Context, tcpConn net.Conn, config *tls.Config) (
	net.Conn, *tls.ConnectionState, error) {
	ev := &TLSHandshakeTrace{
		SourceAddr:    tcpConn.LocalAddr().String(),
		DestAddr:      tcpConn.RemoteAddr().String(),
		SkipTLSVerify: config.InsecureSkipVerify,
		NextProtos:    config.NextProtos,
		StartTime:     time.Now(),
		Error:         nil,
	}
	if net.ParseIP(config.ServerName) == nil {
		ev.ServerName = config.ServerName
	}
	defer tr.add(ev)
	tlsConn, state, err := tr.TLSHandshaker.TLSHandshake(ctx, tcpConn, config)
	ev.EndTime = time.Now()
	ev.Error = err
	if err != nil {
		return nil, nil, err
	}
	ev.Version = state.Version
	ev.CipherSuite = state.CipherSuite
	ev.NegotiatedProto = state.NegotiatedProtocol
	for _, c := range state.PeerCertificates {
		ev.PeerCerts = append(ev.PeerCerts, c.Raw)
	}
	return tlsConn, state, nil
}
