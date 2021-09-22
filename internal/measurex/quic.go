package measurex

//
// QUIC
//
// Wrappers for QUIC to store events into a WritableDB.
//

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICConn is the kind of conn used by QUIC.
type QUICConn = quicx.UDPLikeConn

// QUICDialer creates QUICSesssions.
type QUICDialer = netxlite.QUICDialer

// QUICListener creates listening connections for QUIC.
type QUICListener = netxlite.QUICListener

type quicListenerDB struct {
	netxlite.QUICListener
	begin time.Time
	db    WritableDB
}

func (ql *quicListenerDB) Listen(addr *net.UDPAddr) (QUICConn, error) {
	pconn, err := ql.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return &udpLikeConnDB{
		UDPLikeConn: pconn,
		begin:       ql.begin,
		db:          ql.db,
	}, nil
}

type udpLikeConnDB struct {
	quicx.UDPLikeConn
	begin time.Time
	db    WritableDB
}

func (c *udpLikeConnDB) WriteTo(p []byte, addr net.Addr) (int, error) {
	started := time.Since(c.begin).Seconds()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Operation:  "write_to",
		Network:    "quic",
		RemoteAddr: addr.String(),
		Started:    started,
		Finished:   finished,
		Error:      err,
		Count:      count,
	})
	return count, err
}

func (c *udpLikeConnDB) ReadFrom(b []byte) (int, net.Addr, error) {
	started := time.Since(c.begin).Seconds()
	count, addr, err := c.UDPLikeConn.ReadFrom(b)
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Operation:  "read_from",
		Network:    "quic",
		RemoteAddr: addrStringIfNotNil(addr),
		Started:    started,
		Finished:   finished,
		Error:      err,
		Count:      count,
	})
	return count, addr, err
}

func (c *udpLikeConnDB) Close() error {
	started := time.Since(c.begin).Seconds()
	err := c.UDPLikeConn.Close()
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Operation:  "close",
		Network:    "quic",
		RemoteAddr: "",
		Started:    started,
		Finished:   finished,
		Error:      err,
		Count:      0,
	})
	return err
}

// QUICHandshakeEvent is the result of QUICHandshake.
type QUICHandshakeEvent = TLSHandshakeEvent

// NewQUICDialerWithoutResolver creates a new QUICDialer that is not
// attached to any resolver. This means that every attempt to dial any
// address containing a domain name will fail. This QUICDialer will
// save any event into the WritableDB. Any QUICConn created by it will
// likewise save any event into the WritableDB.
func (mx *Measurer) NewQUICDialerWithoutResolver(db WritableDB, logger Logger) QUICDialer {
	return &quicDialerDB{db: db, logger: logger, begin: mx.Begin}
}

type quicDialerDB struct {
	netxlite.QUICDialer
	begin  time.Time
	db     WritableDB
	logger Logger
}

func (qh *quicDialerDB) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	started := time.Since(qh.begin).Seconds()
	var state tls.ConnectionState
	listener := &quicListenerDB{
		QUICListener: netxlite.NewQUICListener(),
		begin:        qh.begin,
		db:           qh.db,
	}
	dialer := netxlite.NewQUICDialerWithoutResolver(listener, qh.logger)
	defer dialer.CloseIdleConnections()
	sess, err := dialer.DialContext(ctx, network, address, tlsConfig, quicConfig)
	if err == nil {
		select {
		case <-sess.HandshakeComplete().Done():
			state = sess.ConnectionState().TLS.ConnectionState
		case <-ctx.Done():
			sess, err = nil, ctx.Err()
		}
	}
	finished := time.Since(qh.begin).Seconds()
	qh.db.InsertIntoQUICHandshake(&QUICHandshakeEvent{
		Network:         "quic",
		RemoteAddr:      address,
		SNI:             tlsConfig.ServerName,
		ALPN:            tlsConfig.NextProtos,
		SkipVerify:      tlsConfig.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Error:           err,
		Oddity:          qh.computeOddity(err),
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       NewArchivalTLSCerts(peerCerts(nil, &state)),
	})
	return sess, err
}

func (qh *quicDialerDB) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case errorsx.FailureGenericTimeoutError:
		return OddityQUICHandshakeTimeout
	case errorsx.FailureHostUnreachable:
		return OddityQUICHandshakeHostUnreachable
	default:
		return OddityQUICHandshakeOther
	}
}

func (qh *quicDialerDB) CloseIdleConnections() {
	// nothing to do
}
