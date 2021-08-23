package quicdialer

import (
	"errors"
	"net"
	"syscall"
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
	return &saverUDPConn{
		pconn: pconn,
		saver: qls.Saver,
	}, nil
}

type saverUDPConn struct {
	pconn quic.OOBCapablePacketConn
	saver *trace.Saver
}

var (
	_ net.Conn                  = &saverUDPConn{}
	_ quic.OOBCapablePacketConn = &saverUDPConn{}
)

func (c *saverUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	start := time.Now()
	count, err := c.pconn.WriteTo(p, addr)
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

func (c *saverUDPConn) ReadMsgUDP(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
	start := time.Now()
	n, oobn, flags, addr, err := c.pconn.ReadMsgUDP(b, oob)
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

func (c *saverUDPConn) Close() error {
	return c.pconn.Close()
}

func (c *saverUDPConn) LocalAddr() net.Addr {
	// XXX
	conn, ok := c.pconn.(net.Conn)
	if !ok {
		return &net.UDPAddr{}
	}
	return conn.LocalAddr()
}

func (c *saverUDPConn) RemoteAddr() net.Addr {
	// XXX
	conn, ok := c.pconn.(net.Conn)
	if !ok {
		return &net.UDPAddr{}
	}
	return conn.RemoteAddr()
}

func (c *saverUDPConn) Read(b []byte) (int, error) {
	// XXX
	conn, ok := c.pconn.(net.Conn)
	if !ok {
		return 0, errors.New("cannot cast to net.Conn")
	}
	return conn.Read(b)
}

func (c *saverUDPConn) Write(b []byte) (int, error) {
	// XXX
	conn, ok := c.pconn.(net.Conn)
	if !ok {
		return 0, errors.New("cannot cast to net.Conn")
	}
	return conn.Write(b)
}

func (c *saverUDPConn) SetDeadline(d time.Time) error {
	return c.pconn.SetDeadline(d)
}

func (c *saverUDPConn) SetReadDeadline(d time.Time) error {
	return c.pconn.SetReadDeadline(d)
}

func (c *saverUDPConn) SetWriteDeadline(d time.Time) error {
	return c.pconn.SetReadDeadline(d)
}

func (c *saverUDPConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	// XXX
	return c.pconn.ReadFrom(p)
}

func (c *saverUDPConn) SyscallConn() (syscall.RawConn, error) {
	return c.pconn.SyscallConn()
}

func (c *saverUDPConn) WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error) {
	// XXX
	return c.pconn.WriteMsgUDP(b, oob, addr)
}
