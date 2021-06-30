package quicdialer

import (
	"errors"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening PacketConn.
	Listen(addr *net.UDPAddr) (net.PacketConn, error)
}

// QUICListenerSaver is a QUICListener that also implements saving events.
type QUICListenerSaver struct {
	// QUICListener is the underlying QUICListener.
	QUICListener QUICListener

	// Saver is the underlying Saver.
	Saver *trace.Saver
}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerSaver) Listen(addr *net.UDPAddr) (net.PacketConn, error) {
	pconn, err := qls.QUICListener.Listen(addr)
	if err != nil {
		return nil, err
	}
	// TODO(bassosimone): refactor to remove this restriction.
	udpConn, ok := pconn.(*net.UDPConn)
	if !ok {
		return nil, errors.New("quicdialer: cannot convert to udpConn")
	}
	return saverUDPConn{UDPConn: udpConn, saver: qls.Saver}, nil
}

type saverUDPConn struct {
	*net.UDPConn
	saver *trace.Saver
}

func (c saverUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.UDPConn.WriteTo(p, addr)
	stop := time.Now()
	c.saver.Write(trace.Event{
		Address:  addr.String(),
		Data:     p[:count],
		Duration: stop.Sub(start),
		Err:      err,
		NumBytes: count,
		Name:     errorx.WriteToOperation,
		Time:     stop,
	})
	return count, err
}

func (c saverUDPConn) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	start := time.Now()
	n, oobn, flags, addr, err := c.UDPConn.ReadMsgUDP(b, oob)
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
		Name:     errorx.ReadFromOperation,
		Time:     stop,
	})
	return n, oobn, flags, addr, err
}
