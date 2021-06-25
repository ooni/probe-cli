package quicdialer

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// QUICListener listens for QUIC connections.
type QUICListener interface {
	// Listen creates a new listening net.PacketConn.
	Listen(addr *net.UDPAddr) (net.PacketConn, error)
}

// QUICListenerStdlib is a QUICListener using the standard library.
type QUICListenerStdlib struct{}

// Listen implements QUICListener.Listen.
func (qls *QUICListenerStdlib) Listen(addr *net.UDPAddr) (net.PacketConn, error) {
	return net.ListenUDP("udp", addr)
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

// SystemDialer is the basic dialer for QUIC
type SystemDialer struct {
	// QUICListener is the underlying QUICListener to use.
	QUICListener QUICListener
}

// DialContext implements ContextDialer.DialContext
func (d SystemDialer) DialContext(ctx context.Context, network string,
	host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	onlyhost, onlyport, err := net.SplitHostPort(host)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(onlyport)
	if err != nil {
		return nil, err
	}
	ip := net.ParseIP(onlyhost)
	if ip == nil {
		return nil, errors.New("quicdialer: invalid IP representation")
	}
	pconn, err := d.QUICListener.Listen(&net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	udpAddr := &net.UDPAddr{IP: ip, Port: port, Zone: ""}
	return quic.DialEarlyContext(ctx, pconn, udpAddr, host, tlsCfg, cfg)
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
