package netem

//
// UNET: userspace networking.
//

import (
	"context"
	"crypto/x509"
	"net"
	"net/netip"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

// UNetGetaddrinfo is an interface providing the getaddrinfo
// functionality to a given [UNetStack].
//
// The [StaticGetaddrinfo] type implements this interface and can
// be used to provide DNS information, or lack thereof, to OONI Probe
// when running integration tests.
type UNetGetaddrinfo interface {
	// Lookup should behave like [net.LookupHost], except that it should
	// also return the domain's CNAME, if available.
	//
	// This function MUST return [ErrDNSNoSuchHost] in case of NXDOMAIN and
	// [ErrDNSServerMisbehaving] in case of other errors. By doing that, this
	// function will behave exactly like [netxlite]'s getaddrinfo.
	Lookup(ctx context.Context, domain string) (addrs []string, cname string, err error)
}

// UNetStack is a network stack in user space. The zero value is
// nvalid; please, use [NewUNetStack] to construct.
//
// With a [UNetStack] you can create:
//
//   - connected TCP/UDP sockets using [UNetStack.DialContext];
//
//   - listening UDP sockets using [UNetStack.ListenUDP];
//
//   - listening TCP sockets using [UNetStack.ListenTCP];
//
// Because [UNetStack] implements [model.UnderlyingNetwork], you
// can use it to modify the behavior of [netxlite] by using the
// [netxlite.WithCustomTProxy] function.
//
// Because [UNetStack] implements [LinkNIC], a [Link] knows how
// to use it to send and receive [LinkFrame] frames.
type UNetStack struct {
	// closeOnce ensures that Close has "once" semantics.
	closeOnce sync.Once

	// gginfo implements getaddrinfo lookups.
	gginfo UNetGetaddrinfo

	// ipAddress is the IP address we're using.
	ipAddress netip.Addr

	// mtu is the configured MTU.
	mtu uint32

	// name is the name of the NIC.
	name string

	// ns is a network stack in userspace.
	ns *gvisorStack

	// tlsMITMConfig is the MITM config to generate certificates on the fly.
	tlsMITMmConfig *TLSMITMConfig
}

var (
	_ model.UnderlyingNetwork = &UNetStack{}
	_ LinkNIC                 = &UNetStack{}
)

// NewUNetStack constructs a new [UNetStack] instance. This function calls
// [runtimex.PanicOnError] in case of failure.
//
// Arguments:
//
// - A is the IPv4 address to assign to the [UNetStack];
//
// - cfg contains TLS MITM configuration;
//
// - gginfo provides the getaddrinfo functionality to the [UNetStack].
func NewUNetStack(A string, cfg *TLSMITMConfig, gginfo UNetGetaddrinfo) *UNetStack {
	const MTU = 1460
	runtimex.Assert(MTU >= 1300, "MTU too small for using lucas-clemente/quic-go")

	// parse the local address
	addr := runtimex.Try1(netip.ParseAddr(A))

	// create userspace TUN and network stack
	ns := newGVisorStack(addr, MTU)

	// log that we are bringing up a new virtual interface
	name := nextLinkInterfaceID()
	log.Infof("netem: ifconfig %s %s up", name, A)

	// fill and return the network
	return &UNetStack{
		closeOnce:      sync.Once{},
		gginfo:         gginfo,
		ipAddress:      addr,
		name:           name,
		ns:             ns,
		tlsMITMmConfig: cfg,
	}
}

// Close shutds down the virtual network stack.
func (gs *UNetStack) Close() error {
	gs.closeOnce.Do(func() {
		log.Infof("netem: ifconfig %s down", gs.name)
		gs.ns.Close()
	})
	return nil
}

// InterfaceName implements LinkNIC
func (gs *UNetStack) InterfaceName() string {
	return gs.name
}

// IPAddress returns the IP address assigned to the stack.
func (gs *UNetStack) IPAddress() string {
	return gs.ipAddress.String()
}

// ReadPacket implements LinkNIC
func (gs *UNetStack) ReadPacket() ([]byte, error) {
	// create buffer for incoming packet
	const packetbuffer = 1 << 17
	runtimex.Assert(packetbuffer > gs.mtu, "packetbuffer smaller than the MTU")
	buffer := make([]byte, packetbuffer)

	// read incoming packet
	count, err := gs.ns.ReadPacket(buffer)
	if err != nil {
		return nil, err
	}

	// prepare the outgoing frame
	payload := buffer[:count]
	return payload, nil
}

// WritePacket implements LinkNIC
func (gs *UNetStack) WritePacket(packet []byte) error {
	return gs.ns.WritePacket(packet)
}

// DefaultCertPool implements model.UnderlyingNetwork.
func (gs *UNetStack) DefaultCertPool() *x509.CertPool {
	return gs.tlsMITMmConfig.CertPool()
}

// DialContext implements model.UnderlyingNetwork.
func (gs *UNetStack) DialContext(
	ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
	// if needed, configure the timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var (
		conn net.Conn
		err  error
	)

	// parse the address into a [netip.Addr]
	addrport, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, err
	}

	// determine what "dial" actualls means in this context (sorry)
	switch network {
	case "tcp":
		conn, err = gs.ns.DialContextTCPAddrPort(ctx, addrport)
	case "udp":
		conn, err = gs.ns.DialUDPAddrPort(netip.AddrPort{}, addrport)

	default:
		return nil, syscall.EPROTOTYPE
	}

	// make sure we return an error on failure
	if err != nil {
		return nil, mapUNetError(err)
	}

	// wrap returned connection to correctly map errors
	return &unetConnWrapper{conn}, nil
}

// GetaddrinfoLookupANY implements model.UnderlyingNetwork.
func (gs *UNetStack) GetaddrinfoLookupANY(
	ctx context.Context, domain string) ([]string, string, error) {
	return gs.gginfo.Lookup(ctx, domain)
}

// GetaddrinfoResolverNetwork implements model.UnderlyingNetwork
func (gs *UNetStack) GetaddrinfoResolverNetwork() string {
	return "getaddrinfo" // pretend we are calling the getaddrinfo(3) func
}

// ListenUDP implements model.UnderlyingNetwork.
func (gs *UNetStack) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	if network != "udp" {
		return nil, syscall.EPROTOTYPE
	}

	// convert addr to [netip.AddrPort]
	ipaddr, good := netip.AddrFromSlice(addr.IP)
	if !good {
		return nil, syscall.EADDRNOTAVAIL
	}
	addrport := netip.AddrPortFrom(ipaddr, uint16(addr.Port))

	pconn, err := gs.ns.DialUDPAddrPort(addrport, netip.AddrPort{})
	if err != nil {
		return nil, mapUNetError(err)
	}

	return &unetPacketConnWrapper{pconn}, nil
}

// Listen returns a listening TCP connection or an error.
func (gs *UNetStack) ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error) {
	if network != "tcp" {
		return nil, syscall.EPROTOTYPE
	}

	// convert addr to [netip.AddrPort]
	ipaddr, good := netip.AddrFromSlice(addr.IP)
	if !good {
		return nil, syscall.EADDRNOTAVAIL
	}
	addrport := netip.AddrPortFrom(ipaddr, uint16(addr.Port))

	listener, err := gs.ns.ListenTCPAddrPort(addrport)
	if err != nil {
		return nil, mapUNetError(err)
	}

	return &unetListenerWrapper{listener}, nil
}

// unetSuffixToError maps a gvisor error suffix to an stdlib error.
type unetSuffixToError struct {
	// suffix is the unet err.Error() suffix.
	suffix string

	// err is generally a syscall error but it could
	// also be any other stdlib error.
	err error
}

// allUNetSyscallErrors defines [unetSuffixToError] rules for all the
// errors emitted by unet that matter to measuring censorship.
//
// See https://github.com/google/unet/blob/master/pkg/tcpip/errors.go
var allUNetSyscallErrors = []*unetSuffixToError{{
	suffix: "endpoint is closed for receive",
	err:    net.ErrClosed,
}, {
	suffix: "endpoint is closed for send",
	err:    net.ErrClosed,
}, {
	suffix: "connection aborted",
	err:    syscall.ECONNABORTED,
}, {
	suffix: "connection was refused",
	err:    syscall.ECONNREFUSED,
}, {
	suffix: "connection reset by peer",
	err:    syscall.ECONNRESET,
}, {
	suffix: "network is unreachable",
	err:    syscall.ENETUNREACH,
}, {
	suffix: "no route to host",
	err:    syscall.EHOSTUNREACH,
}, {
	suffix: "host is down",
	err:    syscall.EHOSTDOWN,
}, {
	suffix: "machine is not on the network",
	err:    syscall.ENETDOWN,
}, {
	suffix: "operation timed out",
	err:    syscall.ETIMEDOUT,
}}

// mapUNetError maps a unet error to an stdlib error.
func mapUNetError(err error) error {
	if err != nil {
		estring := err.Error()
		for _, entry := range allUNetSyscallErrors {
			if strings.HasSuffix(estring, entry.suffix) {
				return entry.err
			}
		}
	}
	return err
}

// unetConnWrapper wraps a [net.Conn] to remap unet errors
// so that we can emulate stdlib errors.
type unetConnWrapper struct {
	c net.Conn
}

var _ net.Conn = &unetConnWrapper{}

// Close implements net.Conn
func (gcw *unetConnWrapper) Close() error {
	return gcw.c.Close()
}

// LocalAddr implements net.Conn
func (gcw *unetConnWrapper) LocalAddr() net.Addr {
	return gcw.c.LocalAddr()
}

// Read implements net.Conn
func (gcw *unetConnWrapper) Read(b []byte) (n int, err error) {
	count, err := gcw.c.Read(b)
	return count, mapUNetError(err)
}

// RemoteAddr implements net.Conn
func (gcw *unetConnWrapper) RemoteAddr() net.Addr {
	return gcw.c.RemoteAddr()
}

// SetDeadline implements net.Conn
func (gcw *unetConnWrapper) SetDeadline(t time.Time) error {
	return gcw.c.SetDeadline(t)
}

// SetReadDeadline implements net.Conn
func (gcw *unetConnWrapper) SetReadDeadline(t time.Time) error {
	return gcw.c.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn
func (gcw *unetConnWrapper) SetWriteDeadline(t time.Time) error {
	return gcw.c.SetWriteDeadline(t)
}

// Write implements net.Conn
func (gcw *unetConnWrapper) Write(b []byte) (n int, err error) {
	count, err := gcw.c.Write(b)
	return count, mapUNetError(err)
}

// unetPacketConnWrapper wraps a [model.UDPLikeConn] such that we can use
// this connection with lucas-clemente/quic-go and remaps unet errors to
// emulate actual stdlib errors.
type unetPacketConnWrapper struct {
	c *gonet.UDPConn
}

var (
	_ model.UDPLikeConn = &unetPacketConnWrapper{}
	_ syscall.RawConn   = &unetPacketConnWrapper{}
)

// Close implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) Close() error {
	return gpcw.c.Close()
}

// LocalAddr implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) LocalAddr() net.Addr {
	return gpcw.c.LocalAddr()
}

// ReadFrom implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) ReadFrom(p []byte) (int, net.Addr, error) {
	count, addr, err := gpcw.c.ReadFrom(p)
	return count, addr, mapUNetError(err)
}

// SetDeadline implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) SetDeadline(t time.Time) error {
	return gpcw.c.SetDeadline(t)
}

// SetReadDeadline implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) SetReadDeadline(t time.Time) error {
	return gpcw.c.SetReadDeadline(t)
}

// SetWriteDeadline implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) SetWriteDeadline(t time.Time) error {
	return gpcw.c.SetWriteDeadline(t)
}

// WriteTo implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := gpcw.c.WriteTo(p, addr)
	return count, mapUNetError(err)
}

// Implementation note: the following function calls are all stubs and they
// should nonetheless work with lucas-clemente/quic-go.

// SetReadBuffer implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) SetReadBuffer(bytes int) error {
	log.Infof("netem: SetReadBuffer stub called with %d bytes as the argument", bytes)
	return nil
}

// SyscallConn implements model.UDPLikeConn
func (gpcw *unetPacketConnWrapper) SyscallConn() (syscall.RawConn, error) {
	log.Infof("netem: SyscallConn stub called")
	return gpcw, nil
}

// Control implements syscall.RawConn
func (gpcw *unetPacketConnWrapper) Control(f func(fd uintptr)) error {
	log.Infof("netem: Control stub called")
	return nil
}

// Read implements syscall.RawConn
func (gpcw *unetPacketConnWrapper) Read(f func(fd uintptr) (done bool)) error {
	log.Infof("netem: Read stub called")
	return nil
}

// Write implements syscall.RawConn
func (gpcw *unetPacketConnWrapper) Write(f func(fd uintptr) (done bool)) error {
	log.Infof("netem: Write stub called")
	return nil
}

// unetListenerWrapper wraps a [net.Listener] and maps unet
// errors to the corresponding stdlib errors.
type unetListenerWrapper struct {
	l *gonet.TCPListener
}

var _ net.Listener = &unetListenerWrapper{}

// Accept implements net.Listener
func (glw *unetListenerWrapper) Accept() (net.Conn, error) {
	conn, err := glw.l.Accept()
	if err != nil {
		return nil, mapUNetError(err)
	}
	return &unetConnWrapper{conn}, nil
}

// Addr implements net.Listener
func (glw *unetListenerWrapper) Addr() net.Addr {
	return glw.l.Addr()
}

// Close implements net.Listener
func (glw *unetListenerWrapper) Close() error {
	return glw.l.Close()
}
