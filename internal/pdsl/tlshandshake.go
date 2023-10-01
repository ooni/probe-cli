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
func TLSHandshake(ctx context.Context, rt Runtime, tlsConfig *tls.Config) Filter[Result[TCPConn], Result[TLSConn]] {
	return func(mTcpConns <-chan Result[TCPConn]) <-chan Result[TLSConn] {
		outputs := make(chan Result[TLSConn])

		go func() {
			// make sure we close the outputs channel
			defer close(outputs)

			for mTcpConn := range mTcpConns {
				// handle the case of previous stage failure
				if err := mTcpConn.Err; err != nil {
					outputs <- NewResultError[TLSConn](err)
					continue
				}
				tcpConn := mTcpConn.Value
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

				// TLS handshake
				thx := trace.NewTLSHandshakerStdlib(rt.Logger())
				tlsConn, err := thx.Handshake(ctx, tcpConn.Conn, tlsConfig)

				// cancel the context
				cancel()

				// stop the operation logger
				ol.Stop(err)

				// handle failure
				if err != nil {
					outputs <- NewResultError[TLSConn](err)
					continue
				}

				// make sure the Runtime eventually closes the connection
				rt.RegisterCloser(tlsConn)

				// handle success
				outputs <- NewResultValue(TLSConn{trace, tlsConn})
			}
		}()

		return outputs
	}
}
