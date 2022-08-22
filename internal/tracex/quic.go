package tracex

//
// QUIC
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

// QUICDialerSaver saves events occurring during the QUIC handshake.
type QUICDialerSaver struct {
	// QUICDialer is the wrapped dialer
	QUICDialer model.QUICDialer

	// Saver saves events
	Saver *Saver
}

// WrapQUICDialer wraps a model.QUICDialer with a QUICHandshakeSaver that will
// save the QUIC handshake results into this Saver.
//
// When this function is invoked on a nil Saver, it will directly return
// the original QUICDialer without any wrapping.
func (s *Saver) WrapQUICDialer(qd model.QUICDialer) model.QUICDialer {
	if s == nil {
		return qd
	}
	return &QUICDialerSaver{
		QUICDialer: qd,
		Saver:      s,
	}
}

// DialContext implements QUICDialer.DialContext
func (h *QUICDialerSaver) DialContext(ctx context.Context,
	host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
	start := time.Now()
	// TODO(bassosimone): in the future we probably want to also save
	// information about what versions we're willing to accept.
	h.Saver.Write(&EventQUICHandshakeStart{&EventValue{
		Address:       host,
		NoTLSVerify:   tlsCfg.InsecureSkipVerify,
		Proto:         "udp",
		TLSNextProtos: tlsCfg.NextProtos,
		TLSServerName: tlsCfg.ServerName,
		Time:          start,
	}})
	sess, err := h.QUICDialer.DialContext(ctx, host, tlsCfg, cfg)
	stop := time.Now()
	if err != nil {
		// TODO(bassosimone): here we should save the peer certs
		h.Saver.Write(&EventQUICHandshakeDone{&EventValue{
			Address:       host,
			Duration:      stop.Sub(start),
			Err:           NewFailureStr(err),
			NoTLSVerify:   tlsCfg.InsecureSkipVerify,
			Proto:         "udp",
			TLSNextProtos: tlsCfg.NextProtos,
			TLSPeerCerts:  [][]byte{},
			TLSServerName: tlsCfg.ServerName,
			Time:          stop,
		}})
		return nil, err
	}
	state := quicConnectionState(sess)
	h.Saver.Write(&EventQUICHandshakeDone{&EventValue{
		Address:            host,
		Duration:           stop.Sub(start),
		NoTLSVerify:        tlsCfg.InsecureSkipVerify,
		Proto:              "udp",
		TLSCipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		TLSNegotiatedProto: state.NegotiatedProtocol,
		TLSNextProtos:      tlsCfg.NextProtos,
		TLSPeerCerts:       tlsPeerCerts(state, err),
		TLSServerName:      tlsCfg.ServerName,
		TLSVersion:         netxlite.TLSVersionString(state.Version),
		Time:               stop,
	}})
	return sess, nil
}

func (h *QUICDialerSaver) CloseIdleConnections() {
	h.QUICDialer.CloseIdleConnections()
}

// quicConnectionState returns the ConnectionState of a QUIC Session.
func quicConnectionState(sess quic.EarlyConnection) tls.ConnectionState {
	return sess.ConnectionState().TLS.ConnectionState
}

// QUICListenerSaver is a QUICListener that also implements saving events.
type QUICListenerSaver struct {
	// QUICListener is the underlying QUICListener.
	QUICListener model.QUICListener

	// Saver is the underlying Saver.
	Saver *Saver
}

// WrapQUICListener wraps a model.QUICDialer with a QUICListenerSaver that will
// save the QUIC I/O packet conn events into this Saver.
//
// When this function is invoked on a nil Saver, it will directly return
// the original QUICListener without any wrapping.
func (s *Saver) WrapQUICListener(ql model.QUICListener) model.QUICListener {
	if s == nil {
		return ql
	}
	return &QUICListenerSaver{
		QUICListener: ql,
		Saver:        s,
	}
}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerSaver) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	pconn = &quicPacketConnWrapper{
		UDPLikeConn: pconn,
		saver:       qls.Saver,
	}
	return pconn, nil
}

// quicPacketConnWrapper saves I/O events
type quicPacketConnWrapper struct {
	// UDPLikeConn is the wrapped underlying conn
	model.UDPLikeConn

	// Saver saves events
	saver *Saver
}

func (c *quicPacketConnWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	stop := time.Now()
	c.saver.Write(&EventWriteToOperation{&EventValue{
		Address:  addr.String(),
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      NewFailureStr(err),
		NumBytes: count,
		Time:     stop,
	}})
	return count, err
}

func (c *quicPacketConnWrapper) ReadFrom(b []byte) (int, net.Addr, error) {
	start := time.Now()
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	stop := time.Now()
	var data []byte
	if n > 0 {
		data = b[:n]
	}
	c.saver.Write(&EventReadFromOperation{&EventValue{
		Address:  c.safeAddrString(addr),
		Data:     data,
		Duration: stop.Sub(start),
		Err:      NewFailureStr(err),
		NumBytes: n,
		Time:     stop,
	}})
	return n, addr, err
}

func (c *quicPacketConnWrapper) safeAddrString(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}

var _ model.QUICDialer = &QUICDialerSaver{}
var _ model.QUICListener = &QUICListenerSaver{}
var _ model.UDPLikeConn = &quicPacketConnWrapper{}
