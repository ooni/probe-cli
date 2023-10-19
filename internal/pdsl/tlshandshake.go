package pdsl

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TLSConn is the [model.TLSConn] produced by [TLSHandshake].
type TLSConn struct {
	Trace Trace
	model.TLSConn
}

// TCPConnect returns a [Filter] that attempts to create [TLSConn] from [TCPConn].
func TLSHandshake(ctx context.Context, rt Runtime, tlsConfig *tls.Config) Filter[TCPConn, TLSConn] {
	return startFilterService(func(tcpConn TCPConn) (TLSConn, error) {
		sni := tlsConfig.ServerName
		alpn := tlsConfig.NextProtos
		endpoint := tcpConn.RemoteAddr().String()
		trace := tcpConn.Trace

		// start the operation logger
		traceID := rt.NewTraceID()
		ol := logx.NewOperationLogger(
			rt.Logger(),
			"[#%d] TLSHandshake %s SNI=%s ALPN=%s",
			traceID,
			endpoint,
			sni,
			alpn,
		)

		// enforce a timeout
		const timeout = 10 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// TLS handshake
		thx := trace.NewTLSHandshakerStdlib(rt.Logger())
		tlsConn, err := thx.Handshake(ctx, tcpConn.Conn, tlsConfig)

		// stop the operation logger
		ol.Stop(err)

		// handle failure
		if err != nil {
			return TLSConn{}, err
		}

		// make sure the Runtime eventually closes the connection
		rt.RegisterCloser(tlsConn)

		// handle success
		return TLSConn{trace, tlsConn}, nil
	})
}
