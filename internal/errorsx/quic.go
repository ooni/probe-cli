package errorsx

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
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
	Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// ErrorWrapperQUICListener is a QUICListener that wraps errors.
type ErrorWrapperQUICListener struct {
	// QUICListener is the underlying listener.
	QUICListener QUICListener
}

var _ QUICListener = &ErrorWrapperQUICListener{}

// Listen implements QUICListener.Listen.
func (qls *ErrorWrapperQUICListener) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: errorsx.QUICListenOperation,
		}.MaybeBuild()
	}
	return &errorWrapperUDPConn{pconn}, nil
}

// errorWrapperUDPConn is a quicx.UDPLikeConn that wraps errors.
type errorWrapperUDPConn struct {
	// UDPLikeConn is the underlying conn.
	quicx.UDPLikeConn
}

var _ quicx.UDPLikeConn = &errorWrapperUDPConn{}

// WriteTo implements quicx.UDPLikeConn.WriteTo.
func (c *errorWrapperUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Error:     err,
			Operation: errorsx.WriteToOperation,
		}.MaybeBuild()
	}
	return count, nil
}

// ReadFrom implements quicx.UDPLikeConn.ReadFrom.
func (c *errorWrapperUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	if err != nil {
		return 0, nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: errorsx.ReadFromOperation,
		}.MaybeBuild()
	}
	return n, addr, nil
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
		Classifier: errorsx.ClassifyQUICHandshakeError,
		Error:      err,
		Operation:  errorsx.QUICHandshakeOperation,
	}.MaybeBuild()
	if err != nil {
		return nil, err
	}
	return sess, nil
}
