package errorsx

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/quicx"
)

// QUICContextDialer is a dialer for QUIC using Context.
type QUICContextDialer interface {
	// DialContext establishes a new QUIC session using the given
	// network and address. The tlsConfig and the quicConfig arguments
	// MUST NOT be nil. Returns either the session or an error.
	DialContext(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)
}

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quicx.UDPConn, error)
}

// ErrorWrapperQUICListener is a QUICListener that wraps errors.
type ErrorWrapperQUICListener struct {
	// QUICListener is the underlying listener.
	QUICListener QUICListener
}

var _ QUICListener = &ErrorWrapperQUICListener{}

// Listen implements QUICListener.Listen.
func (qls *ErrorWrapperQUICListener) Listen(addr *net.UDPAddr) (quicx.UDPConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: QUICListenOperation,
		}.MaybeBuild()
	}
	return &errorWrapperUDPConn{pconn}, nil
}

// errorWrapperUDPConn is a quicx.UDPConn that wraps errors.
type errorWrapperUDPConn struct {
	// UDPConn is the underlying conn.
	quicx.UDPConn
}

var _ quicx.UDPConn = &errorWrapperUDPConn{}

// WriteTo implements quicx.UDPConn.WriteTo.
func (c *errorWrapperUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPConn.WriteTo(p, addr)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Error:     err,
			Operation: WriteToOperation,
		}.MaybeBuild()
	}
	return count, nil
}

// ReadMsgUDP implements quicx.UDPConn.ReadMsgUDP.
func (c *errorWrapperUDPConn) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	n, oobn, flags, addr, err := c.UDPConn.ReadMsgUDP(b, oob)
	if err != nil {
		return 0, 0, 0, nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: ReadFromOperation,
		}.MaybeBuild()
	}
	return n, oobn, flags, addr, nil
}

// ErrorWrapperQUICDialer is a dialer that performs quic err wrapping
type ErrorWrapperQUICDialer struct {
	Dialer QUICContextDialer
}

// DialContext implements ContextDialer.DialContext
func (d *ErrorWrapperQUICDialer) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	err = SafeErrWrapperBuilder{
		Classifier: ClassifyQUICFailure,
		Error:      err,
		Operation:  QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}
