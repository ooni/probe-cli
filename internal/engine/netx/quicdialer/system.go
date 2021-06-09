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

// SystemDialer is the basic dialer for QUIC
type SystemDialer struct {
	// Saver saves read/write events on the underlying UDP
	// connection. (Implementation note: we need it here since
	// this is the only part in the codebase that is able to
	// observe the underlying UDP connection.)
	Saver *trace.Saver
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
		// TODO(kelmenhorst): write test for this error condition.
		return nil, errors.New("quicdialer: invalid IP representation")
	}
	udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, err
	}
	var pconn net.PacketConn = udpConn
	if d.Saver != nil {
		pconn = saverUDPConn{UDPConn: udpConn, saver: d.Saver}
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
	// TODO(kelmenhorst): 	This is basically the functionality of ErrorWrapperConn.
	// 						Should this be it's own conn wrapper? (we can only access the UDP conn in the system dialer)
	if err != nil {
		err = &ErrWriteTo{err}
	}
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
	// TODO(kelmenhorst): 	This is basically the functionality of ErrorWrapperConn.
	// 						Should this be it's own conn wrapper? (we can only access the UDP conn in the system dialer)
	if err != nil {
		err = &ErrReadFrom{err}
	}
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
