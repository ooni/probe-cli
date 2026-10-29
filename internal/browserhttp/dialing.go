package browserhttp

//
// A transport that connects for each request, for http.Client
//

import (
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DialingTransport connects to the request's host for each request,
// handshakes with [ClientHello] over HTTPS, and sends the request with
// [Transport] and [SetHeaders]' headers. It follows no redirects and keeps
// no connections: use it in an http.Client, which follows redirects.
type DialingTransport struct {
	// Dialer is the MANDATORY dialer, which resolves the host.
	Dialer model.Dialer

	// Handshaker is the MANDATORY TLS handshaker, which should parrot [ClientHello].
	Handshaker model.TLSHandshaker

	// RootCAs is the OPTIONAL pool to verify servers with. If nil, we
	// use netxlite's default pool.
	RootCAs *x509.CertPool
}

// NewDialingTransport returns a [DialingTransport] wrapped with logging
// and error wrapping, resolving names with reso.
func NewDialingTransport(netx *netxlite.Netx, logger model.DebugLogger, reso model.Resolver) model.HTTPTransport {
	return netxlite.WrapHTTPTransport(logger, &DialingTransport{
		Dialer:     netx.NewDialerWithResolver(logger, reso),
		Handshaker: netx.NewTLSHandshakerUTLS(logger, ClientHello),
	})
}

var _ model.HTTPTransport = &DialingTransport{}

// Network implements model.HTTPTransport.
func (txp *DialingTransport) Network() string {
	return "tcp"
}

// CloseIdleConnections implements model.HTTPTransport. We keep none.
func (txp *DialingTransport) CloseIdleConnections() {}

// RoundTrip implements model.HTTPTransport.
func (txp *DialingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	secure := req.URL.Scheme == "https"
	port := req.URL.Port()
	if port == "" {
		port = "80"
		if secure {
			port = "443"
		}
	}
	conn, err := txp.Dialer.DialContext(ctx, "tcp", net.JoinHostPort(req.URL.Hostname(), port))
	if err != nil {
		return nil, err
	}

	alpn := ""
	if secure {
		tlsConn, err := txp.Handshaker.Handshake(ctx, conn, &tls.Config{
			NextProtos: []string{"h2", "http/1.1"},
			RootCAs:    txp.RootCAs,
			ServerName: req.URL.Hostname(),
		})
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		conn = tlsConn
		alpn = netxlite.MaybeTLSConnectionState(tlsConn).NegotiatedProtocol
	}

	// Each hop gets the headers Chrome sends for its scheme, and no Referer,
	// as on the redirects of a typed URL. The client's cookies stay.
	req = req.Clone(ctx)
	req.Header.Del("Referer")
	SetHeaders(req.Header, secure)

	resp, err := NewTransport(conn, alpn).RoundTrip(req)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	resp.Body = &connClosingBody{ReadCloser: resp.Body, conn: conn}
	return resp, nil
}

// connClosingBody closes the connection with the body.
type connClosingBody struct {
	io.ReadCloser
	conn net.Conn
}

func (b *connClosingBody) Close() error {
	err := b.ReadCloser.Close()
	_ = b.conn.Close()
	return err
}
