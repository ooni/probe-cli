package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"

	"github.com/lucas-clemente/quic-go"
)

// QUICDialerContext is a dialer for QUIC using Context.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// QUICDialer dials QUIC connections.
type QUICDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	Dial(network, address string, tlsConfig *tls.Config,
		quicConfig *quic.Config) (quic.EarlySession, error)
}

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening PacketConn.
	Listen(addr *net.UDPAddr) (net.PacketConn, error)
}

// QUICListenerStdlib is a QUICListener using the standard library.
type QUICListenerStdlib struct{}

var _ QUICListener = &QUICListenerStdlib{}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerStdlib) Listen(addr *net.UDPAddr) (net.PacketConn, error) {
	return net.ListenUDP("udp", addr)
}

// QUICDialerQUICGo dials using the lucas-clemente/quic-go library.
type QUICDialerQUICGo struct {
	// QUICListener is the underlying QUICListener to use.
	QUICListener QUICListener
}

var _ QUICContextDialer = &QUICDialerQUICGo{}

// errInvalidIP indicates that a string is not a valid IP.
var errInvalidIP = errors.New("netxlite: invalid IP")

// DialContext implements ContextDialer.DialContext
func (d *QUICDialerQUICGo) DialContext(ctx context.Context, network string,
	address string, tlsConfig *tls.Config, quicConfig *quic.Config) (
	quic.EarlySession, error) {
	onlyhost, onlyport, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(onlyport)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(onlyhost)
	if ip == nil {
		return nil, errInvalidIP
	}
	pconn, err := d.QUICListener.Listen(&net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	udpAddr := &net.UDPAddr{IP: ip, Port: port, Zone: ""}
	sess, err := quic.DialEarlyContext(
		ctx, pconn, udpAddr, address, tlsConfig, quicConfig)
	if err != nil {
		return nil, err
	}
	return &quicSessionOwnsConn{EarlySession: sess, conn: pconn}, nil
}

// quicSessionOwnsConn ensures that we close the PacketConn.
type quicSessionOwnsConn struct {
	quic.EarlySession

	// conn is the connection we own
	conn net.PacketConn
}

// CloseWithError implements quic.EarlySession.CloseWithError.
func (sess *quicSessionOwnsConn) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	err := sess.EarlySession.CloseWithError(code, reason)
	sess.conn.Close()
	return err
}
