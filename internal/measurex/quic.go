package measurex

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

// QUICListener creates listening connections for QUIC.
type QUICListener = netxlite.QUICListener

// WrapQUICListener takes in input a netxlite.QUICListener and returns
// a new listener that saves measurements into the DB.
func WrapQUICListener(origin Origin, db EventDB, ql netxlite.QUICListener) QUICListener {
	return &quicListenerx{
		QUICListener: ql,
		db:           db,
		origin:       origin,
	}
}

type quicListenerx struct {
	netxlite.QUICListener
	db     EventDB
	origin Origin
}

// QUICPacketConn is an UDP PacketConn used by QUIC.
type QUICPacketConn = quicx.UDPLikeConn

func (ql *quicListenerx) Listen(addr *net.UDPAddr) (QUICPacketConn, error) {
	pconn, err := ql.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return &quicUDPLikeConnx{
		UDPLikeConn: pconn,
		connID:      ql.db.NextConnID(),
		db:          ql.db,
		localAddr:   pconn.LocalAddr().String(),
		origin:      ql.origin,
	}, nil
}

type quicUDPLikeConnx struct {
	quicx.UDPLikeConn
	connID    int64
	db        EventDB
	localAddr string
	origin    Origin
}

func (c *quicUDPLikeConnx) WriteTo(p []byte, addr net.Addr) (int, error) {
	started := c.db.ElapsedTime()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	finished := c.db.ElapsedTime()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "write_to",
		Network:       string(NetworkQUIC),
		RemoteAddr:    addr.String(),
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         count,
	})
	return count, err
}

func (c *quicUDPLikeConnx) ReadFrom(b []byte) (int, net.Addr, error) {
	started := c.db.ElapsedTime()
	count, addr, err := c.UDPLikeConn.ReadFrom(b)
	finished := c.db.ElapsedTime()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "read_from",
		Network:       string(NetworkQUIC),
		RemoteAddr:    c.addrStringIfNotNil(addr),
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         count,
	})
	return count, addr, err
}

func (c *quicUDPLikeConnx) addrStringIfNotNil(addr net.Addr) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}

func (c *quicUDPLikeConnx) Close() error {
	started := c.db.ElapsedTime()
	err := c.UDPLikeConn.Close()
	finished := c.db.ElapsedTime()
	c.db.InsertIntoReadWrite(&NetworkEvent{
		Origin:        c.origin,
		MeasurementID: c.db.MeasurementID(),
		ConnID:        c.connID,
		Operation:     "close",
		Network:       string(NetworkQUIC),
		RemoteAddr:    "",
		LocalAddr:     c.localAddr,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Count:         0,
	})
	return err
}

// LocalAddr returns the local address and also implements a
// hack to pass to the session the ConnID.
func (c *quicUDPLikeConnx) LocalAddr() net.Addr {
	localAddr := c.UDPLikeConn.LocalAddr()
	if localAddr == nil {
		return nil
	}
	return &quicLocalAddrx{Addr: localAddr, connID: c.connID}
}

type quicLocalAddrx struct {
	net.Addr
	connID int64
}

// QUICEarlySession is the type we use to wrap quic.EarlySession. This
// kind of session knows about the underlying ConnID.
type QUICEarlySession interface {
	quic.EarlySession

	ConnID() int64
}

// QUICDialer creates QUIC sessions. This kind of dialer will
// save QUIC handshake measurements into the DB.
type QUICDialer interface {
	DialContext(ctx context.Context, address string,
		tlsConfig *tls.Config) (QUICEarlySession, error)

	CloseIdleConnections()
}

// QUICHandshakeEvent is the result of QUICHandshake.
type QUICHandshakeEvent struct {
	Origin          Origin
	MeasurementID   int64
	ConnID          int64
	Network         string
	RemoteAddr      string
	LocalAddr       string
	SNI             string
	ALPN            []string
	SkipVerify      bool
	Started         time.Duration
	Finished        time.Duration
	Error           error
	Oddity          Oddity
	TLSVersion      string
	CipherSuite     string
	NegotiatedProto string
	PeerCerts       [][]byte
}

// WrapQUICDialer creates a new QUICDialer that will save
// QUIC handshake events into the DB.
func WrapQUICDialer(origin Origin, db EventDB, dialer netxlite.QUICDialer) QUICDialer {
	return &quicDialerx{
		QUICDialer: dialer,
		origin:     origin,
		db:         db,
	}
}

type quicDialerx struct {
	netxlite.QUICDialer
	db     EventDB
	origin Origin
}

func (qh *quicDialerx) DialContext(ctx context.Context,
	address string, tlsConfig *tls.Config) (QUICEarlySession, error) {
	started := qh.db.ElapsedTime()
	var (
		localAddr *quicLocalAddrx
		state     tls.ConnectionState
	)
	sess, err := qh.QUICDialer.DialContext(
		ctx, "udp", address, tlsConfig, &quic.Config{})
	if err == nil {
		select {
		case <-sess.HandshakeComplete().Done():
			state = sess.ConnectionState().TLS.ConnectionState
			if addr := sess.LocalAddr(); addr != nil {
				if laddr, ok := addr.(*quicLocalAddrx); ok {
					localAddr = laddr
				}
			}
		case <-ctx.Done():
			sess, err = nil, ctx.Err()
		}
	}
	finished := qh.db.ElapsedTime()
	qh.db.InsertIntoQUICHandshake(&QUICHandshakeEvent{
		Origin:          qh.origin,
		MeasurementID:   qh.db.MeasurementID(),
		ConnID:          qh.connIDIfNotNil(localAddr),
		Network:         string(NetworkQUIC),
		RemoteAddr:      address,
		LocalAddr:       qh.localAddrIfNotNil(localAddr),
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
		PeerCerts:       peerCerts(nil, &state),
	})
	if err != nil {
		return nil, err
	}
	return &quicEarlySessionx{
		EarlySession: sess, connID: qh.connIDIfNotNil(localAddr)}, nil
}

func (qh *quicDialerx) computeOddity(err error) Oddity {
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

func (qh *quicDialerx) connIDIfNotNil(addr *quicLocalAddrx) (out int64) {
	if addr != nil {
		out = addr.connID
	}
	return
}

func (qh *quicDialerx) localAddrIfNotNil(addr *quicLocalAddrx) (out string) {
	if addr != nil {
		out = addr.String()
	}
	return
}

func (qh *quicDialerx) CloseIdleConnections() {
	qh.QUICDialer.CloseIdleConnections()
}

type quicEarlySessionx struct {
	quic.EarlySession
	connID int64
}

func (qes *quicEarlySessionx) ConnID() int64 {
	return qes.connID
}
