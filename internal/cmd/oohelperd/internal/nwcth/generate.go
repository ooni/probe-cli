package nwcth

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Generate is the third step of the algorithm. Given the
// observed round trips, we generate measurement targets and
// execute those measurements so the probe has a benchmark.

// Generator is the interface responsible for running Generate.
type Generator interface {
	Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error)
}

// DefaultGenerator is the default Generator.
type DefaultGenerator struct {
	dialer     netxlite.Dialer
	quicDialer netxlite.QUICContextDialer
	resolver   netxlite.Resolver
}

// Generate takes in input a list of round trips and outputs
// a list of connectivity measurements for each of them.
func (g *DefaultGenerator) Generate(ctx context.Context, rts []*RoundTrip) ([]*URLMeasurement, error) {
	var out []*URLMeasurement
	for _, rt := range rts {
		currentURL := g.GenerateURL(ctx, rt)
		out = append(out, currentURL)
	}
	return out, nil
}

// GenerateURL returns a URLMeasurement.
func (g *DefaultGenerator) GenerateURL(ctx context.Context, rt *RoundTrip) *URLMeasurement {
	addrs, err := g.DNSDo(ctx, rt.Request.URL.Hostname())
	currentURL := &URLMeasurement{
		DNS: &DNSMeasurement{
			Domain:  rt.Request.URL.Hostname(),
			Addrs:   addrs,
			Failure: newfailure(err),
		},
		RoundTrip: rt,
		URL:       rt.Request.URL.String(),
	}
	if err != nil {
		return currentURL
	}
	for _, addr := range addrs {
		var port string
		explicitPort := rt.Request.URL.Port()
		scheme := rt.Request.URL.Scheme
		switch {
		case explicitPort != "":
			port = explicitPort
		case scheme == "http":
			port = "80"
		case scheme == "https":
			port = "443"
		default:
			panic("should not happen")
		}
		endpoint := net.JoinHostPort(addr, port)

		var currentEndpoint EndpointMeasurement
		_, h3 := supportedQUICVersions[rt.proto]
		switch {
		case h3:
			currentEndpoint = g.GenerateH3Endpoint(ctx, rt, endpoint)
		case rt.proto == "http":
			currentEndpoint = g.GenerateHTTPEndpoint(ctx, rt, endpoint)
		case rt.proto == "https":
			currentEndpoint = g.GenerateHTTPSEndpoint(ctx, rt, endpoint)
		default:
			// TODO(kelmenhorst): do we have to register this error somewhere in the result struct?
			continue
		}
		currentURL.Endpoints = append(currentURL.Endpoints, currentEndpoint)
	}
	return currentURL
}

// GenerateHTTPEndpoint performs an HTTP Request by
// a) establishing a TCP connection to the target (TCPDo),
// b) performing an HTTP GET request to the endpoint (HTTPDo).
// It returns an EndpointMeasurement.
func (g *DefaultGenerator) GenerateHTTPEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPEndpointMeasurement{
		Endpoint: endpoint,
	}
	tcpConn, err := g.TCPDo(ctx, endpoint)
	currentEndpoint.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	defer tcpConn.Close()

	// prepare HTTPRoundtripMeasurement of this endpoint
	currentEndpoint.HTTPRoundtripMeasurement = &HTTPRoundtripMeasurement{
		Request: &HTTPRequest{
			Headers: rt.Request.Header,
		},
	}
	transport := NewSingleTransport(tcpConn)
	resp, body, err := g.HTTPDo(rt.Request, transport)
	if err != nil {
		// failed Response
		currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
			Failure: newfailure(err),
		}
		return currentEndpoint
	}
	// successful Response
	currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return currentEndpoint
}

// GenerateHTTPSEndpoint performs an HTTPS Request by
// a) establishing a TCP connection to the target (TCPDo),
// b) establishing a TLS connection to the target (TLSDo),
// c) performing an HTTP GET request to the endpoint (HTTPDo).
// It returns an EndpointMeasurement.
func (g *DefaultGenerator) GenerateHTTPSEndpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &HTTPSEndpointMeasurement{
		Endpoint: endpoint,
	}
	var tcpConn, tlsConn net.Conn
	tcpConn, err := g.TCPDo(ctx, endpoint)
	currentEndpoint.TCPConnectMeasurement = &TCPConnectMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	defer tcpConn.Close()

	tlsConn, err = g.TLSDo(tcpConn, rt.Request.URL.Hostname())
	currentEndpoint.TLSHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	defer tlsConn.Close()

	// prepare HTTPRoundtripMeasurement of this endpoint
	currentEndpoint.HTTPRoundtripMeasurement = &HTTPRoundtripMeasurement{
		Request: &HTTPRequest{
			Headers: rt.Request.Header,
		},
	}
	transport := NewSingleTransport(tlsConn)
	resp, body, err := g.HTTPDo(rt.Request, transport)
	if err != nil {
		// failed Response
		currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
			Failure: newfailure(err),
		}
		return currentEndpoint
	}
	// successful Response
	currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return currentEndpoint
}

// GenerateH3Endpoint performs an HTTP/3 Request by
// a) establishing a QUIC connection to the target (QUICDo),
// b) performing an HTTP GET request to the endpoint (HTTPDo).
// It returns an EndpointMeasurement.
func (g *DefaultGenerator) GenerateH3Endpoint(ctx context.Context, rt *RoundTrip, endpoint string) EndpointMeasurement {
	currentEndpoint := &H3EndpointMeasurement{
		Endpoint: endpoint,
	}
	tlsConf := &tls.Config{
		ServerName: rt.Request.URL.Hostname(),
		NextProtos: []string{rt.proto},
	}
	sess, err := g.QUICDo(ctx, endpoint, tlsConf)
	currentEndpoint.QUICHandshakeMeasurement = &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
	if err != nil {
		return currentEndpoint
	}
	// prepare HTTPRoundtripMeasurement of this endpoint
	currentEndpoint.HTTPRoundtripMeasurement = &HTTPRoundtripMeasurement{
		Request: &HTTPRequest{
			Headers: rt.Request.Header,
		},
	}
	transport := NewSingleH3Transport(sess, tlsConf, &quic.Config{})
	resp, body, err := g.HTTPDo(rt.Request, transport)
	if err != nil {
		// failed Response
		currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
			Failure: newfailure(err),
		}
		return currentEndpoint
	}
	// successful Response
	currentEndpoint.HTTPRoundtripMeasurement.Response = &HTTPResponse{
		BodyLength: int64(len(body)),
		Failure:    nil,
		Headers:    resp.Header,
		StatusCode: int64(resp.StatusCode),
	}
	return currentEndpoint
}

// DNSDo performs the DNS check.
func (g *DefaultGenerator) DNSDo(ctx context.Context, domain string) ([]string, error) {
	return g.resolver.LookupHost(ctx, domain)
}

// TCPDo performs the TCP check.
func (g *DefaultGenerator) TCPDo(ctx context.Context, endpoint string) (net.Conn, error) {
	if g.dialer != nil {
		return g.dialer.DialContext(ctx, "tcp", endpoint)
	}
	dialer := NewDialerResolver(g.resolver)
	return dialer.DialContext(ctx, "tcp", endpoint)
}

// TLSDo performs the TLS check.
func (g *DefaultGenerator) TLSDo(conn net.Conn, hostname string) (*tls.Conn, error) {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: hostname,
		NextProtos: []string{"h2", "http/1.1"},
	})
	err := tlsConn.Handshake()
	return tlsConn, err
}

// QUICDo performs the QUIC check.
func (g *DefaultGenerator) QUICDo(ctx context.Context, endpoint string, tlsConf *tls.Config) (quic.EarlySession, error) {
	if g.quicDialer != nil {
		return g.quicDialer.DialContext(ctx, "udp", endpoint, tlsConf, &quic.Config{})
	}
	dialer := NewQUICDialerResolver(g.resolver)
	return dialer.DialContext(ctx, "udp", endpoint, tlsConf, &quic.Config{})
}

// HTTPDo performs the HTTP check.
func (g *DefaultGenerator) HTTPDo(req *http.Request, transport http.RoundTripper) (*http.Response, []byte, error) {
	clnt := http.Client{
		CheckRedirect: func(r *http.Request, reqs []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
	}
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp, nil, nil
	}
	return resp, body, nil
}
