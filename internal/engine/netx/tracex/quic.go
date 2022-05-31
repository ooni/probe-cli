package tracex

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// QUICHandshakeSaver saves events occurring during the handshake
type QUICHandshakeSaver struct {
	Saver *Saver
	model.QUICDialer
}

// DialContext implements ContextDialer.DialContext
func (h QUICHandshakeSaver) DialContext(ctx context.Context, network string,
	host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
	start := time.Now()
	// TODO(bassosimone): in the future we probably want to also save
	// information about what versions we're willing to accept.
	h.Saver.Write(Event{
		Address:       host,
		Name:          "quic_handshake_start",
		NoTLSVerify:   tlsCfg.InsecureSkipVerify,
		Proto:         network,
		TLSNextProtos: tlsCfg.NextProtos,
		TLSServerName: tlsCfg.ServerName,
		Time:          start,
	})
	sess, err := h.QUICDialer.DialContext(ctx, network, host, tlsCfg, cfg)
	stop := time.Now()
	if err != nil {
		h.Saver.Write(Event{
			Duration:      stop.Sub(start),
			Err:           err,
			Name:          "quic_handshake_done",
			NoTLSVerify:   tlsCfg.InsecureSkipVerify,
			TLSNextProtos: tlsCfg.NextProtos,
			TLSServerName: tlsCfg.ServerName,
			Time:          stop,
		})
		return nil, err
	}
	state := quicConnectionState(sess)
	h.Saver.Write(Event{
		Duration:           stop.Sub(start),
		Name:               "quic_handshake_done",
		NoTLSVerify:        tlsCfg.InsecureSkipVerify,
		TLSCipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		TLSNegotiatedProto: state.NegotiatedProtocol,
		TLSNextProtos:      tlsCfg.NextProtos,
		TLSPeerCerts:       PeerCerts(state, err),
		TLSServerName:      tlsCfg.ServerName,
		TLSVersion:         netxlite.TLSVersionString(state.Version),
		Time:               stop,
	})
	return sess, nil
}

// quicConnectionState returns the ConnectionState of a QUIC Session.
func quicConnectionState(sess quic.EarlyConnection) tls.ConnectionState {
	return sess.ConnectionState().TLS.ConnectionState
}

// QUICListenerSaver is a QUICListener that also implements saving events.
type QUICListenerSaver struct {
	// QUICListener is the underlying QUICListener.
	model.QUICListener

	// Saver is the underlying Saver.
	Saver *Saver
}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerSaver) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return &saverUDPConn{
		UDPLikeConn: pconn,
		saver:       qls.Saver,
	}, nil
}

type saverUDPConn struct {
	model.UDPLikeConn
	saver *Saver
}

var _ model.UDPLikeConn = &saverUDPConn{}

func (c *saverUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	stop := time.Now()
	c.saver.Write(Event{
		Address:  addr.String(),
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     netxlite.WriteToOperation,
		Time:     stop,
	})
	return count, err
}

func (c *saverUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	start := time.Now()
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	stop := time.Now()
	var data []byte
	if n > 0 {
		data = b[:n]
	}
	c.saver.Write(Event{
		Address:  c.safeAddrString(addr),
		Data:     data,
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: n,
		Name:     netxlite.ReadFromOperation,
		Time:     stop,
	})
	return n, addr, err
}

func (c *saverUDPConn) safeAddrString(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}
