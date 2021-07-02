package quicdialer

import (
	"net"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening UDPConn.
	Listen(addr *net.UDPAddr) (quic.OOBCapablePacketConn, error)
}

// QUICListenerSaver is a QUICListener that also implements saving events.
type QUICListenerSaver struct {
	// QUICListener is the underlying QUICListener.
	QUICListener QUICListener

	// Saver is the underlying Saver.
	Saver *trace.Saver
}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerSaver) Listen(addr *net.UDPAddr) (quic.OOBCapablePacketConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	return saverUDPConn{OOBCapablePacketConn: pconn, saver: qls.Saver}, nil
}

type saverUDPConn struct {
	quic.OOBCapablePacketConn
	saver *trace.Saver
}

func (c saverUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.OOBCapablePacketConn.WriteTo(p, addr)
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

func (c saverUDPConn) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	start := time.Now()
	n, oobn, flags, addr, err := c.OOBCapablePacketConn.ReadMsgUDP(b, oob)
	stop := time.Now()
	var data []byte
	if n > 0 {
		data = b[:n]
	}
	c.saver.Write(trace.Event{
		Address:  addr.String(),
		Data:     data,
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: n,
		Name:     errorsx.ReadFromOperation,
		Time:     stop,
	})
	return n, oobn, flags, addr, err
}
