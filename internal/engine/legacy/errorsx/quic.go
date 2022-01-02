package errorsx

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	Listen(addr *net.UDPAddr) (model.UDPLikeConn, error)
}

// ErrorWrapperQUICListener is a QUICListener that wraps errors.
type ErrorWrapperQUICListener struct {
	// QUICListener is the underlying listener.
	QUICListener QUICListener
}

var _ QUICListener = &ErrorWrapperQUICListener{}

// Listen implements QUICListener.Listen.
func (qls *ErrorWrapperQUICListener) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: netxlite.QUICListenOperation,
		}.MaybeBuild()
	}
	return &errorWrapperUDPConn{pconn}, nil
}

// errorWrapperUDPConn is a model.UDPLikeConn that wraps errors.
type errorWrapperUDPConn struct {
	// UDPLikeConn is the underlying conn.
	model.UDPLikeConn
}

var _ model.UDPLikeConn = &errorWrapperUDPConn{}

// WriteTo implements model.UDPLikeConn.WriteTo.
func (c *errorWrapperUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	if err != nil {
		return 0, SafeErrWrapperBuilder{
			Error:     err,
			Operation: netxlite.WriteToOperation,
		}.MaybeBuild()
	}
	return count, nil
}

// ReadFrom implements model.UDPLikeConn.ReadFrom.
func (c *errorWrapperUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	n, addr, err := c.UDPLikeConn.ReadFrom(b)
	if err != nil {
		return 0, nil, SafeErrWrapperBuilder{
			Error:     err,
			Operation: netxlite.ReadFromOperation,
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
	if err != nil {
		return nil, SafeErrWrapperBuilder{
			Classifier: netxlite.ClassifyQUICHandshakeError,
			Error:      err,
			Operation:  netxlite.QUICHandshakeOperation,
		}.MaybeBuild()
	}
	return sess, nil
}
