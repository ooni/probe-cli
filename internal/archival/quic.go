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
	network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	started := time.Now()
	var state tls.ConnectionState
	sess, err := dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err == nil {
		<-sess.HandshakeComplete().Done() // robustness (the dialer already does that)
		state = sess.ConnectionState().TLS.ConnectionState
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
