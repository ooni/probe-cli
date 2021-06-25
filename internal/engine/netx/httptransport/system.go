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

// NewSystemTransport creates a new "system" HTTP transport. That is a transport
// using the Go standard library with custom dialer and TLS dialer.
func NewSystemTransport(config Config) RoundTripper {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	txp.DialContext = config.Dialer.DialContext
	txp.DialTLSContext = config.TLSDialer.DialTLSContext
	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	txp.MaxConnsPerHost = 1
	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	txp.DisableCompression = true
	return newTransport(txp, config)
}

var _ RoundTripper = &http.Transport{}

func newTransport(txp *http.Transport, config Config) RoundTripper {
	return &roundTripper{underlyingTransport: txp, tlsdialer: config.TLSDialer, tlsconfig: config.TLSConfig}
}

// roundTripper is a wrapper around the system transport
type roundTripper struct {
	sync.Mutex
	ctx                 context.Context
	scheme              string
	tlsconfig           *tls.Config
	tlsdialer           TLSDialer
	transport           RoundTripper // this will be either http.Transport or http2.Transport
	underlyingTransport *http.Transport
}

func (rt *roundTripper) CloseIdleConnections() {
	rt.underlyingTransport.CloseIdleConnections()
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.scheme != strings.ToLower(req.URL.Scheme) {
		rt.transport = nil
	}
	if rt.transport == nil {
		// determine transport type to use for this Roundtrip
		if err := rt.getTransport(req); err != nil {
			return nil, err
		}
	}
	return rt.transport.RoundTrip(req)
}

var errTransportCreated = errors.New("used ALPN to determine transport type")

func (rt *roundTripper) getTransport(req *http.Request) error {
	rt.scheme = strings.ToLower(req.URL.Scheme)
	switch rt.scheme {
	case "http":
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
		return rt.tlsdialer.DialTLSContext(ctx, network, addr)
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
	// TLS handshake
	tlsconn := tls.Client(conn, cfg)
	err = tlsconn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}
	// use ALPN to decide which Transport to use
	switch tlsconn.ConnectionState().NegotiatedProtocol {
	case "http/1.1":
		// HTTP 1.x + TLS.
		rt.transport = rt.underlyingTransport
	default:
		// assume HTTP 2 + TLS.
		rt.ctx = ctx // there is no DialTLSContext in http2.Transport so we have to remember it in roundTripper
		rt.transport = &http2.Transport{
			DialTLS:            rt.dialTLSHTTP2,
			DisableCompression: rt.underlyingTransport.DisableCompression,
		}
	}
	return nil, errTransportCreated
}

// dialTLSHTTP2 fits the signature of http2.Transport.DialTLS
func (rt *roundTripper) dialTLSHTTP2(network, addr string, cfg *tls.Config) (net.Conn, error) {
	rt.tlsconfig = cfg
	return rt.dialTLSContext(rt.ctx, network, addr)
}

func getDialTLSAddr(u *url.URL) string {
	host, port, err := net.SplitHostPort(u.Host)
	if err == nil {
		return net.JoinHostPort(host, port)
	}
	return net.JoinHostPort(u.Host, u.Scheme)
}
