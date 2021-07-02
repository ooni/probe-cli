package httptransport

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"golang.org/x/net/http2"
)

func newRoundtripper(txp *http.Transport, tlsdialer TLSDialer, tlsconfig *tls.Config) RoundTripper {
	return &roundTripper{underlyingTransport: txp, DialTLS: tlsdialer.DialTLSContext, tlsconfig: tlsconfig}
}

// roundTripper is a wrapper around the system transport
type roundTripper struct {
	sync.Mutex
	ctx                 context.Context
	DialTLS             func(ctx context.Context, network string, address string) (net.Conn, error)
	tlsconfig           *tls.Config
	transport           http.RoundTripper // this will be either http.Transport or http2.Transport
	underlyingTransport *http.Transport
}

func (rt *roundTripper) CloseIdleConnections() {
	rt.underlyingTransport.CloseIdleConnections()
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	// determine transport type to use for this Roundtrip
	rt.Lock()
	rt.transport = nil
	rt.Unlock()

	if err := rt.getTransport(req); err != nil {
		return nil, err
	}
	return rt.transport.RoundTrip(req)
}

var errTransportCreated = errors.New("used ALPN to determine transport type")

func (rt *roundTripper) getTransport(req *http.Request) error {
	scheme := strings.ToLower(req.URL.Scheme)
	switch scheme {
	case "http":
		// HTTP 1.x w/o TLS
		rt.transport = rt.underlyingTransport
		return nil
	case "https":
	default:
		return errors.New("invalid scheme")
	}
	ctx := req.Context()
	_, err := rt.dialTLSContext(ctx, "tcp", getDialTLSAddr(req.URL))
	switch err {
	case errTransportCreated: // intended behavior
		return nil
	case nil:
		return errors.New("dialTLS returned no error when determining transport")
	default:
		return err
	}
}

func (rt *roundTripper) dialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	rt.Lock() // we are updating state
	defer rt.Unlock()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	if rt.transport != nil {
		// transport is already determined: use standard DialTLSContext
		return rt.DialTLS(ctx, network, addr)
	}
	// connect
	conn, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	// set TLS config
	cfg := rt.tlsconfig
	if cfg == nil {
		cfg = new(tls.Config)
	}
	if cfg.ServerName == "" {
		cfg.ServerName = host
	}
	cfg.NextProtos = []string{"h2", "http/1.1"}

	// TLS handshake
	tlsconn := tls.Client(conn, cfg)
	err = tlsconn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}
	// use ALPN to decide which Transport to use
	switch tlsconn.ConnectionState().NegotiatedProtocol {
	case "h2":
		// HTTP 2 + TLS.
		rt.ctx = ctx // there is no DialTLSContext in http2.Transport so we have to remember it in roundTripper
		rt.transport = &http2.Transport{
			DialTLS:            rt.dialTLSHTTP2,
			TLSClientConfig:    rt.underlyingTransport.TLSClientConfig,
			DisableCompression: rt.underlyingTransport.DisableCompression,
		}
	default:
		// assume HTTP 1.x + TLS.
		rt.transport = rt.underlyingTransport
	}
	return nil, errTransportCreated
}

// dialTLSHTTP2 fits the signature of http2.Transport.DialTLS
func (rt *roundTripper) dialTLSHTTP2(network, addr string, cfg *tls.Config) (net.Conn, error) {
	return rt.dialTLSContext(rt.ctx, network, addr)
}

func getDialTLSAddr(u *url.URL) string {
	host, port, err := net.SplitHostPort(u.Host)
	if err == nil {
		return net.JoinHostPort(host, port)
	}
	return net.JoinHostPort(u.Host, u.Scheme)
}
