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
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type quicListenerDB struct {
	model.QUICListener
	begin time.Time
	db    WritableDB
}

func (ql *quicListenerDB) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
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
	model.UDPLikeConn
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
		Failure:    NewFailure(err),
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
		Failure:    NewFailure(err),
		Count:      count,
	})
	return count, addr, err
}

func (c *udpLikeConnDB) Close() error {
	started := time.Since(c.begin).Seconds()
	err := c.UDPLikeConn.Close()
	finished := time.Since(c.begin).Seconds()
	c.db.InsertIntoClose(&NetworkEvent{
		Operation:  "close",
		Network:    "quic",
		RemoteAddr: "",
		Started:    started,
		Finished:   finished,
		Failure:    NewFailure(err),
		Count:      0,
	})
	return err
}

// NewQUICDialerWithoutResolver creates a new QUICDialer that is not
// attached to any resolver. This means that every attempt to dial any
// address containing a domain name will fail. This QUICDialer will
// save any event into the WritableDB. Any QUICConn created by it will
// likewise save any event into the WritableDB.
func (mx *Measurer) NewQUICDialerWithoutResolver(db WritableDB, logger model.Logger) model.QUICDialer {
	return &quicDialerDB{db: db, logger: logger, begin: mx.Begin}
}

type quicDialerDB struct {
	model.QUICDialer
	begin  time.Time
	db     WritableDB
	logger model.Logger
}

func (qh *quicDialerDB) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
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
		<-sess.HandshakeComplete().Done() // robustness (the dialer already does that)
		state = sess.ConnectionState().TLS.ConnectionState
	}
	finished := time.Since(qh.begin).Seconds()
	qh.db.InsertIntoQUICHandshake(&QUICTLSHandshakeEvent{
		Network:         "quic",
		RemoteAddr:      address,
		SNI:             tlsConfig.ServerName,
		ALPN:            tlsConfig.NextProtos,
		SkipVerify:      tlsConfig.InsecureSkipVerify,
		Started:         started,
		Finished:        finished,
		Failure:         NewFailure(err),
		Oddity:          qh.computeOddity(err),
		TLSVersion:      netxlite.TLSVersionString(state.Version),
		CipherSuite:     netxlite.TLSCipherSuiteString(state.CipherSuite),
		NegotiatedProto: state.NegotiatedProtocol,
		PeerCerts:       peerCerts(nil, &state),
	})
	return sess, err
}

func (qh *quicDialerDB) computeOddity(err error) Oddity {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case netxlite.FailureGenericTimeoutError:
		return OddityQUICHandshakeTimeout
	case netxlite.FailureHostUnreachable:
		return OddityQUICHandshakeHostUnreachable
	default:
		return OddityQUICHandshakeOther
	}
}

func (qh *quicDialerDB) CloseIdleConnections() {
	// nothing to do
}
