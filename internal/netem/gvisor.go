package netem

//
// GVisor-based userspace network stack.
//
// Adapted from https://github.com/WireGuard/wireguard-go
//
// SPDX-License-Identifier: MIT
//

import (
	"context"
	"errors"
	"net"
	"net/netip"
	"os"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"gvisor.dev/gvisor/pkg/bufferv2"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

// gvisorStack is a TCP/IP stack in userspace. Seen from above this
// stack allows creating TCP and UDP connections. Seen from below, it
// allows one to read and write IP packets. The zero value of this
// structure is invalid; please, use [newGVisorStack] to instantiate.
type gvisorStack struct {
	// closeOnce ensures that Close has once semantics.
	closeOnce sync.Once

	// closed is closed by Close and signals that we should
	// not perform any further TCP/IP operation.
	closed chan any

	// endpoint is the endpoint receiving gvisor notifications.
	endpoint *channel.Endpoint

	// incomingPacket is the channel posted by GVisor
	// when there is an incoming IP packet.
	incomingPacket chan *bufferv2.View

	// stack is the network stack in userspace.
	stack *stack.Stack
}

// newGVisorStack creates a new [gvisorStack] instance using A as
// its IPv4 address and MTU as the maximum transfer unit. This
// constructor calls [runtimex.PanicOnError] in case of failure.
func newGVisorStack(A netip.Addr, MTU uint32) *gvisorStack {

	// create options for the new stack
	stackOptions := stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
			ipv6.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
		},
		HandleLocal: true,
	}

	// create the stack instance
	gvs := &gvisorStack{
		closeOnce:      sync.Once{},
		closed:         make(chan any),
		endpoint:       channel.New(1024, MTU, ""),
		incomingPacket: make(chan *bufferv2.View),
		stack:          stack.New(stackOptions),
	}

	// register network as the notification target for gvisor
	gvs.endpoint.AddNotify(gvs)

	// create a NIC to attach to this stack
	gvisorTry0(gvs.stack.CreateNIC(1, gvs.endpoint))

	// configure the IPv4 address for the NIC we created
	protoAddr := tcpip.ProtocolAddress{
		Protocol:          ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.Address(A.AsSlice()).WithPrefix(),
	}
	gvisorTry0(gvs.stack.AddProtocolAddress(1, protoAddr, stack.AddressProperties{}))

	// install the IPv4 address in the routing table
	gvs.stack.AddRoute(tcpip.Route{Destination: header.IPv4EmptySubnet, NIC: 1})

	return gvs
}

// ReadPacket blocks until there is an incoming IPv4 packet or the
// userspace TCP/IP stack is closed.
func (gvs *gvisorStack) ReadPacket(packet []byte) (int, error) {
	select {
	case <-gvs.closed:
		return 0, os.ErrClosed
	case view := <-gvs.incomingPacket:
		return view.Read(packet)
	}
}

// WriteNotify implements channel.Notification. GVisor will call this
// callback function everytime there's a new readable packet.
func (gvs *gvisorStack) WriteNotify() {
	pkt := gvs.endpoint.Read()
	if pkt.IsNil() {
		return
	}
	view := pkt.ToView()
	pkt.DecRef()
	select {
	case gvs.incomingPacket <- view:
	case <-gvs.closed:
	}
}

func (gvs *gvisorStack) WritePacket(packet []byte) error {
	// there is clearly a race condition with closing but the intent is just
	// to behave and return ErrClose long after we've been closed
	select {
	case <-gvs.closed:
		return net.ErrClosed
	default:
	}

	// the following code is already ready for supporting IPv6
	// should we want to do that in the future
	pkb := stack.NewPacketBuffer(stack.PacketBufferOptions{Payload: bufferv2.MakeWithData(packet)})
	switch packet[0] >> 4 {
	case 4:
		gvs.endpoint.InjectInbound(header.IPv4ProtocolNumber, pkb)
	case 6:
		gvs.endpoint.InjectInbound(header.IPv6ProtocolNumber, pkb)
	}

	return nil
}

// Close ensures that we cannot send and recv additional packets and
// that we cannot establish new TCP/UDP connections.
func (gvs *gvisorStack) Close() error {
	gvs.closeOnce.Do(func() {
		// synchronize with other users (MUST be first)
		close(gvs.closed)

		// tear down the gvisor userspace stack
		gvs.endpoint.Close()
		gvs.stack.RemoveNIC(1)
	})
	return nil
}

// DialContextTCPAddrPort establishes a new TCP connection.
func (gvs *gvisorStack) DialContextTCPAddrPort(
	ctx context.Context, addr netip.AddrPort) (*gonet.TCPConn, error) {
	fa, pn := gvisorConvertToFullAddr(addr)
	return gonet.DialContextTCP(ctx, gvs.stack, fa, pn)
}

// ListenTCPAddrPort creates a new listening TCP socket.
func (gvs *gvisorStack) ListenTCPAddrPort(addr netip.AddrPort) (*gonet.TCPListener, error) {
	fa, pn := gvisorConvertToFullAddr(addr)
	return gonet.ListenTCP(gvs.stack, fa, pn)
}

// DialUDPAddrPort allows to create UDP sockets. Using a nil
// raddr is equivalent to [net.ListenUDP]. using nil laddr instead
// is equivalent to [net.DialContext] with an "udp" network.
func (gvs *gvisorStack) DialUDPAddrPort(laddr, raddr netip.AddrPort) (*gonet.UDPConn, error) {
	var lfa, rfa *tcpip.FullAddress
	var pn tcpip.NetworkProtocolNumber

	if laddr.IsValid() || laddr.Port() > 0 {
		var addr tcpip.FullAddress
		addr, pn = gvisorConvertToFullAddr(laddr)
		lfa = &addr
	}

	if raddr.IsValid() || raddr.Port() > 0 {
		var addr tcpip.FullAddress
		addr, pn = gvisorConvertToFullAddr(raddr)
		rfa = &addr
	}

	return gonet.DialUDP(gvs.stack, lfa, rfa, pn)
}

// gvisorConvertToFullAddr is a convenience function for converting
// a [netip.AddrPort] to the kind of addrs used by GVisor.
func gvisorConvertToFullAddr(endpoint netip.AddrPort) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	var protoNumber tcpip.NetworkProtocolNumber

	// the following code is already ready for supporting IPv6
	// should we want to do that in the future
	if endpoint.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	} else {
		protoNumber = ipv6.ProtocolNumber
	}

	fa := tcpip.FullAddress{
		NIC:  1,
		Addr: tcpip.Address(endpoint.Addr().AsSlice()),
		Port: endpoint.Port(),
	}

	return fa, protoNumber
}

// gvisorTry0 emulates [runtimex.Try0] for GVisor-specific errors.
func gvisorTry0(err tcpip.Error) {
	if err != nil {
		runtimex.PanicOnError(errors.New(err.String()), "Try0")
	}
}
