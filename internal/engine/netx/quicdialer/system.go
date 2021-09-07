package quicdialer

import (
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// QUICListenerSaver is a QUICListener that also implements saving events.
type QUICListenerSaver struct {
	// QUICListener is the underlying QUICListener.
	QUICListener QUICListener

	// Saver is the underlying Saver.
	Saver *trace.Saver
}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerSaver) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
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
	quicx.UDPLikeConn
	saver *trace.Saver
}

var _ quicx.UDPLikeConn = &saverUDPConn{}

func (c *saverUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.UDPLikeConn.WriteTo(p, addr)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Address:  addr.String(),
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorsx.WriteToOperation,
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
	c.saver.Write(trace.Event{
		Address:  c.safeAddrString(addr),
		Data:     data,
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: n,
		Name:     errorsx.ReadFromOperation,
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
