package main

import (
	"context"
	"net"
	"net/netip"
	"syscall"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

// maybeHijackNetworkOperations replaces the underlying network operations
// to move traffic to a given peer UDP socket rather than performing network
// operations locally. For this hijacking to happen, the [tproxy] argument
// must be a nonempty string. Otherwise, this function is a no-op.
func maybeHijackNetworkOperations(tproxy string) {
	if tproxy == "" {
		return
	}
	net := newNetstackNet(tproxy)
	hj := &hijacker{net}
	netxlite.TProxyDialWithDialer = hj.dialWithDialer
	netxlite.TProxyGetaddrinfoLookupANY = hj.getaddrinfoLookupANY
	netxlite.TProxyListenUDP = hj.listenUDP
}

// hijacker hijacks low-level network operations.
type hijacker struct {
	net *netstack.Net
}

// newNetstackNet constructs a new instance of netstack.Net.
func newNetstackNet(tproxy string) *netstack.Net {
	// the following code has been adapted from ooni/minivpn
	localSocket, err := net.Dial("tcp", tproxy)
	runtimex.PanicOnError(err, "net.ListenUDP failed")
	const conservativeMTU = 1250
	tun, net, err := netstack.CreateNetTUN(
		[]netip.Addr{
			netip.Addr(netip.MustParseAddr("10.17.17.4")),
		},
		[]netip.Addr{
			netip.MustParseAddr("10.17.17.1"),
		},
		conservativeMTU,
	)
	runtimex.PanicOnError(err, "netstack.CreateNetTun failed")
	go hijackerRoutingLoop(localSocket, tun)
	return net
}

// hijackerRoutingLoop routes traffic between [localSocket] and [tun].
func hijackerRoutingLoop(localSocket net.Conn, tun tun.Device) {
	// the following code has been adapted from ooni/minivpn
	const zeroOffset = 0
	go func() {
		buf := make([]byte, 4096)
		for {
			count, err := tun.Read(buf, zeroOffset)
			if err != nil {
				log.Errorf("hijack: tun read error: %v", err)
				break
			}
			frame := buf[:count]
			if err := netxlite.WriteSimpleFrame(localSocket, frame); err != nil {
				log.Errorf("hijack: localSocket write error: %v", err)
				break
			}
		}
	}()
	go func() {
		for {
			frame, err := netxlite.ReadSimpleFrame(localSocket)
			if err != nil {
				log.Errorf("hijack: localSocket read error: %v", err)
				break
			}
			if _, err = tun.Write(frame, zeroOffset); err != nil {
				log.Errorf("hijack: tun write error: %v", err)
				break
			}
		}
	}()
}

// dialWithDialer replaces netxlite.TProxyDialWithDialer
func (hj *hijacker) dialWithDialer(ctx context.Context, d *net.Dialer, network string, address string) (net.Conn, error) {
	if d.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, d.Timeout)
		defer cancel()
	}
	return hj.net.DialContext(ctx, network, address)
}

// listenUDP replaces netxlite.TProxyListenUDP
func (hj *hijacker) listenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := hj.net.ListenUDP(addr)
	if err != nil {
		return nil, err
	}
	pwrap := &hijackerUDPConn{pconn}
	return pwrap, nil
}

// hijackerUDPConn adapts net.PacketConn to model.UDPLikeConn.
//
// The lucas-clemente/quic-go library is happier if we implement a
// model.UDPLikeConn and our codebase currently tries to make it
// happy very explicitly. This struct and hijackerRawConn are our
// attempts at making quic-go happy also with hijacking. We're still
// getting warnings about impossibility to increase buffer sizes,
// but, at least, we're able to I/O QUIC packets.
type hijackerUDPConn struct {
	net.PacketConn
}

// SetReadBuffer allows setting the read buffer.
func (c *hijackerUDPConn) SetReadBuffer(bytes int) error {
	return nil
}

// SyscallConn returns a conn suitable for calling syscalls,
// which is also instrumental to setting the read buffer.
func (c *hijackerUDPConn) SyscallConn() (syscall.RawConn, error) {
	return &hijackerRawConn{}, nil
}

// hijackerRawConn implements syscall.RawConn
type hijackerRawConn struct{}

// Control implements syscall.RawConn
func (*hijackerRawConn) Control(f func(fd uintptr)) error {
	return nil
}

// Read implements syscall.RawConn
func (*hijackerRawConn) Read(f func(fd uintptr) (done bool)) error {
	return nil
}

// Write implements syscall.RawConn
func (*hijackerRawConn) Write(f func(fd uintptr) (done bool)) error {
	return nil
}

// getaddrinfoLookupANY replaces netxlite.TProxyGetaddrinfoANY
func (hj *hijacker) getaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	addrs, err := hj.net.LookupContextHost(ctx, domain)
	return addrs, "", err
}
