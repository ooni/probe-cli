package measurexlite

//
// QUIC tracing
//

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WrapQUICListener returns a wrapped model.QUICListener that uses this trace.
func (tx *Trace) WrapQUICListener(listener model.QUICListener) model.QUICListener {
	return &quicListenerTrace{
		QUICListener: listener,
		tx:           tx,
	}
}

// quicListenerTrace is a trace-aware QUIC listener.
type quicListenerTrace struct {
	model.QUICListener
	tx *Trace
}

// Listen implements model.QUICListener.Listen
func (ql *quicListenerTrace) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := ql.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return ql.tx.WrapUDPLikeConn(pconn), nil
}

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
func (qdx *quicDialerTrace) DialContext(ctx context.Context, network string,
	address string, tlsConfig *tls.Config, quicConfig *quic.Config) (
	quic.EarlyConnection, error) {
	return qdx.qd.DialContext(netxlite.ContextWithTrace(ctx, qdx.tx), network, address, tlsConfig, quicConfig)
}

// CloseIdleConnections implements model.QUICDialer.CloseIdleConnections.
func (qdx *quicDialerTrace) CloseIdleConnections() {
	qdx.qd.CloseIdleConnections()
}

// OnQUICHandshakeStart implements model.Trace.OnQUICHandshakeStart
func (tx *Trace) OnQUICHandshakeStart(now time.Time, remoteAddr string, config *quic.Config) {
	t := now.Sub(tx.ZeroTime)
	select {
	case tx.NetworkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "quic_handshake_start"):
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
	case tx.QUICHandshake <- NewArchivalTLSOrQUICHandshakeResult(
		tx.Index,
		started.Sub(tx.ZeroTime),
		"quic",
		remoteAddr,
		config,
		state,
		err,
		t,
	):
	default: // buffer is full
	}
	select {
	case tx.NetworkEvent <- NewAnnotationArchivalNetworkEvent(tx.Index, t, "quic_handshake_done"):
	default: // buffer is full
	}
}

// QUICHandshakes drains the network events buffered inside the QUICHandshake channel.
func (tx *Trace) QUICHandshakes() (out []*model.ArchivalTLSOrQUICHandshakeResult) {
	for {
		select {
		case ev := <-tx.QUICHandshake:
			out = append(out, ev)
		default:
			return // done
		}
	}
}
