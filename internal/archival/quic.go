package archival

//
// Saves QUIC events.
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

// WriteTo performs WriteTo with the given pconn and saves the
// operation's results inside the saver.
func (s *Saver) WriteTo(pconn model.UDPLikeConn, buf []byte, addr net.Addr) (int, error) {
	started := time.Now()
	count, err := pconn.WriteTo(buf, addr)
	s.appendNetworkEvent(&NetworkEvent{
		Count:      count,
		Failure:    err,
		Finished:   time.Now(),
		Network:    addr.Network(),
		Operation:  netxlite.WriteToOperation,
		RemoteAddr: addr.String(),
		Started:    started,
	})
	return count, err
}

// ReadFrom performs ReadFrom with the given pconn and saves the
// operation's results inside the saver.
func (s *Saver) ReadFrom(pconn model.UDPLikeConn, buf []byte) (int, net.Addr, error) {
	started := time.Now()
	count, addr, err := pconn.ReadFrom(buf)
	s.appendNetworkEvent(&NetworkEvent{
		Count:      count,
		Failure:    err,
		Finished:   time.Now(),
		Network:    "udp", // must be always set even on failure
		Operation:  netxlite.ReadFromOperation,
		RemoteAddr: s.safeAddrString(addr),
		Started:    started,
	})
	return count, addr, err
}

func (s *Saver) safeAddrString(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}

// QUICDialContext dials a QUIC session using the given dialer
// and saves the results inside of the saver.
func (s *Saver) QUICDialContext(ctx context.Context, dialer model.QUICDialer,
	network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	started := time.Now()
	var state tls.ConnectionState
	sess, err := dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err == nil {
		select {
		case <-sess.HandshakeComplete().Done():
			state = sess.ConnectionState().TLS.ConnectionState
		case <-ctx.Done():
			sess, err = nil, ctx.Err()
		}
	}
	s.appendQUICHandshake(&QUICTLSHandshakeEvent{
		ALPN:            tlsConfig.NextProtos,
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		Failure:         err,
		Finished:        time.Now(),
		NegotiatedProto: state.NegotiatedProtocol,
		Network:         "quic",
		PeerCerts:       s.tlsPeerCerts(err, &state),
		RemoteAddr:      address,
		SNI:             tlsConfig.ServerName,
		SkipVerify:      tlsConfig.InsecureSkipVerify,
		Started:         started,
		TLSVersion:      netxlite.TLSVersionString(state.Version),
	})
	return sess, err
}

func (s *Saver) appendQUICHandshake(ev *QUICTLSHandshakeEvent) {
	s.mu.Lock()
	s.trace.QUICHandshake = append(s.trace.QUICHandshake, ev)
	s.mu.Unlock()
}

// WrapQUICDialer takes in input a QUIC dialer and returns in
// output a new one using this saver to save events.
func (s *Saver) WrapQUICDialer(d model.QUICDialer) model.QUICDialer {
	return &quicDialerSaver{QUICDialer: d, s: s}
}

type quicDialerSaver struct {
	model.QUICDialer
	s *Saver
}

func (d *quicDialerSaver) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	return d.s.QUICDialContext(ctx, d.QUICDialer, network, address, tlsConfig, quicConfig)
}

// WrapQUICListener takes in input a QUIC listener and returns
// in output a new one using this saver to save events.
func (s *Saver) WrapQUICListener(ql model.QUICListener) model.QUICListener {
	return &quicListenerSaver{QUICListener: ql, s: s}
}

type quicListenerSaver struct {
	model.QUICListener
	s *Saver
}

func (ql *quicListenerSaver) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := ql.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	pconn = &quicListenerUDPLikeConn{
		UDPLikeConn: pconn,
		s:           ql.s,
	}
	return pconn, nil
}

type quicListenerUDPLikeConn struct {
	model.UDPLikeConn
	s *Saver
}

func (c *quicListenerUDPLikeConn) ReadFrom(buf []byte) (int, net.Addr, error) {
	return c.s.ReadFrom(c.UDPLikeConn, buf)
}

func (c *quicListenerUDPLikeConn) WriteTo(buf []byte, addr net.Addr) (int, error) {
	return c.s.WriteTo(c.UDPLikeConn, buf, addr)
}
