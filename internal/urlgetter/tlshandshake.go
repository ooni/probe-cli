package urlgetter

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TLSHandshake measures a tlshandshake://<domain>:<port>/ URL.
func (rx *Runner) TLSHandshake(ctx context.Context, config *Config, URL *url.URL) error {
	conn, err := rx.tlsHandshake(ctx, config, URL)
	measurexlite.MaybeClose(conn)
	return err
}

func (rx *Runner) tlsHandshake(ctx context.Context, config *Config, URL *url.URL) (*TLSConn, error) {
	// resolve the URL's domain using DNS
	addrs, err := rx.DNSLookupOp(ctx, config, URL)
	if err != nil {
		return nil, err
	}

	// loop until we establish a single TLS connection
	var errv []error
	for _, addr := range addrs {
		conn, err := rx.TCPConnectOp(ctx, addr)
		if err != nil {
			errv = append(errv, err)
			continue
		}
		tlsconn, err := rx.TLSHandshakeOp(ctx, conn)
		if err != nil {
			conn.Close()
			errv = append(errv, err)
			continue
		}
		return tlsconn, nil
	}

	// either return a joined error or nil
	return nil, errors.Join(errv...)
}

// TLSConn is an established TLS connection.
type TLSConn struct {
	// Config is the original config.
	Config *Config

	// Conn is the conn.
	Conn model.TLSConn

	// Trace is the trace.
	Trace *measurexlite.Trace

	// URL is the original URL.
	URL *url.URL
}

var _ io.Closer = &TLSConn{}

// AsHTTPConn converts a [*TCPConn] to an [*HTTPConn].
func (cx *TLSConn) AsHTTPConn(logger model.Logger) *HTTPConn {
	return &HTTPConn{
		Config:                cx.Config,
		Conn:                  cx.Conn,
		Network:               "tcp",
		RemoteAddress:         measurexlite.SafeRemoteAddrString(cx.Conn),
		TLSNegotiatedProtocol: cx.Conn.ConnectionState().NegotiatedProtocol,
		Trace:                 cx.Trace,
		Transport: netxlite.NewHTTPTransportWithOptions(
			logger,
			netxlite.NewNullDialer(),
			netxlite.NewSingleUseTLSDialer(cx.Conn),
		),
		URL: cx.URL,
	}
}

// Close implements io.Closer.
func (tx *TLSConn) Close() error {
	return tx.Conn.Close()
}

func (cx *Config) alpns(URL *url.URL, httpsValues []string) []string {
	// handle the case where the user explicitly provided ALPNs
	if len(cx.TLSNextProtos) > 0 {
		return strings.Split(cx.TLSNextProtos, ",")
	}

	// otherwise try to use the scheme.
	switch URL.Scheme {
	case "https":
		return httpsValues
	case "dot":
		return []string{"dot"}
	default:
		return nil
	}
}

func (cx *Config) sni(URL *url.URL) string {
	// handle the case where there's an explicit SNI
	if len(cx.TLSServerName) > 0 {
		return cx.TLSServerName
	}

	// otheriwse use the URL's hostname.
	return URL.Hostname()
}

// TLSHandshakeOp performs TLS handshakes.
func (rx *Runner) TLSHandshakeOp(ctx context.Context, input *TCPConn) (*TLSConn, error) {
	// enforce timeout
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// obtain logger
	logger := rx.Session.Logger()

	// obtain the ALPNs
	alpns := input.Config.alpns(input.URL, []string{"h2", "http/1.1"})

	// obtain the SNI
	serverName := input.Config.sni(input.URL)

	// start operation logger
	ol := logx.NewOperationLogger(
		logger,
		"[#%d] TLS handshake ALPN=%v SNI=%s",
		input.Trace.Index(),
		alpns,
		serverName,
	)

	// create the TLS handshaker to use
	tlsHandshaker := input.Trace.NewTLSHandshakerStdlib(logger)

	// creae the TLS config to use
	//
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	tlsConfig := &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		NextProtos: alpns,
		RootCAs:    nil,
		ServerName: serverName,
	}

	// perform the handshake
	tlsConn, err := tlsHandshaker.Handshake(ctx, input.Conn, tlsConfig)

	// stop the operation logger
	ol.Stop(err)

	// append the TLS handshake results
	rx.TestKeys.AppendTLSHandshakes(input.Trace.TLSHandshakes()...)

	// handle the case of failure
	if err != nil {
		// make sure we close the connection
		input.Conn.Close()

		// make sure we set failure and failed operation
		rx.TestKeys.MaybeSetFailedOperation(netxlite.TLSHandshakeOperation)
		rx.TestKeys.MaybeSetFailure(err.Error())
		return nil, err
	}

	// handle the case of success
	result := &TLSConn{
		Config: input.Config,
		Conn:   tlsConn,
		Trace:  input.Trace,
		URL:    input.URL,
	}
	return result, nil
}
