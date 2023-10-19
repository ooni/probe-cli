package pdsl

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/quic-go/quic-go"
)

// QUICConn is the [model.QUICConn] produced by [QUICHandshake].
type QUICConn struct {
	Trace Trace
	quic.EarlyConnection
}

// TCPConnect returns a [Filter] that attempts to create [QUICConn] from [TCPConn].
func QUICHandshake(ctx context.Context, rt Runtime, tlsConfig *tls.Config, tags ...string) Filter[Endpoint, QUICConn] {
	return startFilterService(func(endpoint Endpoint) (QUICConn, error) {
		sni := tlsConfig.ServerName
		alpn := tlsConfig.NextProtos

		// start operation logger
		traceID := rt.NewTraceID()
		ol := logx.NewOperationLogger(
			rt.Logger(),
			"[#%d] QUICHandshake %s SNI=%s ALPN=%s",
			traceID,
			endpoint,
			sni,
			alpn,
		)

		// create trace for collecting OONI observations
		trace := rt.NewTrace(traceID, rt.ZeroTime(), tags...)

		// enforce a timeout
		const timeout = 10 * time.Second
		ctx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		// QUIC handshake
		udpListener := netxlite.NewUDPListener()
		thx := trace.NewQUICDialerWithoutResolver(udpListener, rt.Logger())
		quicConn, err := thx.DialContext(ctx, string(endpoint), tlsConfig, &quic.Config{})

		// stop the operation logger
		ol.Stop(err)

		// handle failure
		if err != nil {
			return QUICConn{}, err
		}

		// make sure the Runtime eventually closes the connection
		rt.RegisterCloser(&quicCloserConn{quicConn})

		// handle success
		return QUICConn{trace, quicConn}, nil
	})
}

type quicCloserConn struct {
	quic.EarlyConnection
}

func (c *quicCloserConn) Close() error {
	return c.CloseWithError(0, "")
}
