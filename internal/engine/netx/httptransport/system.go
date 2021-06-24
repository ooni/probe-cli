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

	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
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

func newTransport(txp *http.Transport, config Config) RoundTripper {
	return &roundTripper{underlyingTransport: txp, config: config}
}

var _ RoundTripper = &http.Transport{}

// TODO(kelmenhorst): restructure and make this code modular
type roundTripper struct {
	sync.Mutex
	underlyingTransport *http.Transport
	transport           http.RoundTripper
	config              Config
	ctx                 context.Context
	initConn            net.Conn
	scheme              string
}

func (rt *roundTripper) CloseIdleConnections() {
	rt.underlyingTransport.CloseIdleConnections()
}

func (rt *roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.scheme != strings.ToLower(req.URL.Scheme) {
		rt.transport = nil
	}
	if rt.transport == nil {
		if err := rt.getTransport(req); err != nil {
			return nil, err
		}
	}
	return rt.transport.RoundTrip(req)
}

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
	case errAbort:
	case nil:
		return errors.New("dialTLS returned no error when determining transport")
	default:
		return err
	}
	return nil
}

var errAbort = errors.New("protocol negotiated")

func (rt *roundTripper) dialTLSContext(ctx context.Context, network, addr string) (net.Conn, error) {
	// Unlike rt.transport, this is protected by a critical section
	// since past the initial manual call from getTransport, the HTTP
	// client will be the caller.
	rt.Lock()
	defer rt.Unlock()

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}
	tlsdial := rt.config.TLSDialer.(tlsdialer.TLSDialer)
	var conn net.Conn
	if rt.transport != nil {
		return tlsdial.DialTLSContext(ctx, network, addr)
	}
	conn, err = net.Dial(network, addr)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	if err != nil {
		return nil, err
	}
	tlsconfig := rt.config.TLSConfig
	if tlsconfig == nil {
		tlsconfig = new(tls.Config)
	} else {
		tlsconfig = tlsconfig.Clone()
	}
	if tlsconfig.ServerName == "" {
		tlsconfig.ServerName = host
	}
	tlsconn := tls.Client(conn, tlsconfig)
	err = tlsconn.Handshake()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if rt.transport != nil {
		return tlsconn, nil
	}
	state := tlsconn.ConnectionState()

	// No http.Transport constructed yet, create one based on the results
	// of ALPN.
	switch state.NegotiatedProtocol {
	case "http/1.1":
		rt.ctx = ctx
		rt.transport = &http.Transport{DialTLSContext: rt.dialTLSContext}
		// The remote peer is speaking HTTP 1.x + TLS.
	default:
		// Assume the remote peer is speaking HTTP 2 + TLS.
		rt.ctx = ctx
		rt.transport = &http2.Transport{DialTLS: rt.dialTLSHTTP2}
	}

	// Stash the connection just established for use servicing the
	// actual request (should be near-immediate).
	rt.initConn = tlsconn

	return nil, errAbort
}

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
