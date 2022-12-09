package measurexlite

//
// QUIC tracing
//

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewQUICDialerWithoutResolver is equivalent to netxlite.NewQUICDialerWithoutResolver
// except that it returns a model.QUICDialer that uses this trace.
func (tx *Trace) NewQUICDialerWithoutResolver(listener model.QUICListener, dl model.DebugLogger) model.QUICDialer {
	return &quicDialerTrace{
		qd: tx.newQUICDialerWithoutResolver(listener, dl),
		tx: tx,
	}
}

// quicDialerTrace is a trace-aware QUIC dialer.
type quicDialerTrace struct {
	qd model.QUICDialer
	tx *Trace
}

var _ model.QUICDialer = &quicDialerTrace{}

// DialContext implements model.QUICDialer.DialContext.
func (qdx *quicDialerTrace) DialContext(ctx context.Context,
	address string, tlsConfig *tls.Config, quicConfig *quic.Config) (
	quic.EarlyConnection, error) {
	return qdx.qd.DialContext(netxlite.ContextWithTrace(ctx, qdx.tx), address, tlsConfig, quicConfig)
}

// CloseIdleConnections implements model.QUICDialer.CloseIdleConnections.
func (qdx *quicDialerTrace) CloseIdleConnections() {
	qdx.qd.CloseIdleConnections()
}

// OnQUICHandshakeStart implements model.Trace.OnQUICHandshakeStart
func (tx *Trace) OnQUICHandshakeStart(now time.Time, remoteAddr string, config *quic.Config) {
	t := now.Sub(tx.ZeroTime)
	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "quic_handshake_start"):
	default:
	}
}

// OnQUICHandshakeDone implements model.Trace.OnQUICHandshakeDone
func (tx *Trace) OnQUICHandshakeDone(started time.Time, remoteAddr string, qconn quic.EarlyConnection,
	config *tls.Config, err error, finished time.Time) {
	t := finished.Sub(tx.ZeroTime)
	state := tls.ConnectionState{}
	if qconn != nil {
		state = qconn.ConnectionState().TLS.ConnectionState
	}
	select {
	case tx.quicHandshake <- NewArchivalTLSOrQUICHandshakeResult(
		tx.Index,
		started.Sub(tx.ZeroTime),
		"udp",
		remoteAddr,
		config,
		state,
		err,
		t,
	):
	default: // buffer is full
	}
	select {
	case tx.networkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "quic_handshake_done"):
	default: // buffer is full
	}
}

// QUICHandshakes drains the network events buffered inside the QUICHandshake channel.
func (tx *Trace) QUICHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult) {
	for {
		select {
		case ev := <-tx.quicHandshake:
			out = append(out, ev)
		default:
			return // done
		}
	}
}

// FirstQUICHandshakeOrNil drains the network events buffered inside the QUICHandshake channel
// and returns the first QUICHandshake, if any. Otherwise, it returns nil.
func (tx *Trace) FirstQUICHandshakeOrNil() *model.ArchivalTLSOrQUICHandshakeResult {
	ev := tx.QUICHandshakes()
	if len(ev) < 1 {
		return nil
	}
	return ev[0]
}

// MaybeCloseQUICConn is a convenience function for closing a quic.EarlyConnection only when such a conn
// isn't nil.
func MaybeCloseQUICConn(conn quic.EarlyConnection) (err error) {
	if conn != nil {
		err = conn.CloseWithError(0, "")
	}
	return
}
