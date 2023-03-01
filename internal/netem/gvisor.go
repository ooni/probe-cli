package netem

//
// Gvisor- and wireguard- based networking in userspace.
//

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

// GvisorGetaddrinfo is an interface providing the getaddrinfo
// functionality to a given [GvisorStack].
//
// The [StaticGetaddrinfo] type implements this interface and can
// be used to provide DNS information, or lack thereof, to OONI Probe
// when running integration tests.
type GvisorGetaddrinfo interface {
	// Lookup should behave like net.LookupHost, except that it should
	// also return the domain CNAME, if available.
	//
	// This function MUST return [ErrDNSNoSuchHost] in case of NXDOMAIN and
	// [ErrDNSServerMisbehaving] in case of other errors. By doing that, this
	// function will behave exactly like [netxlite]'s getaddrinfo.
	Lookup(ctx context.Context, domain string) (addrs []string, cname string, err error)
}

// GvisorStack is a network stack in user space. The zero value is invalid;
// please, use [NewGvisorStack] to construct a new instance.
//
// With a [GvisorStack] you can create:
//
//   - connected TCP/UDP sockets using [GvisorStack.DialContext];
//
//   - listening UDP sockets using [GvisorStack.ListenUDP];
//
//   - listening TCP sockets using [GvisorStack.ListenTCP];
//
// Because [GvisorStack] implements [model.UnderlyingNetwork], you
// can use it to modify the behavior of [netxlite] by using the
// [netxlite.WithCustomTProxy] function.
//
// However, a [GvisorStack] does not route traffic anywhere. To do
// that, you need to connect it with a [NIC] by using the
// [GvisorStack.Attach] function.
type GvisorStack struct {
	// gginfo implements getaddrinfo lookups.
	gginfo GvisorGetaddrinfo

	// ipAddress is the IP address we're using.
	ipAddress netip.Addr

	// netStack is a network stack in user space.
	netStack *netstack.Net

	// tlsMITMConfig is the MITM config to generate certificates on the fly.
	tlsMITMmConfig *TLSMITMConfig

	// tunDevice is the userspace TUN device.
	tunDevice tun.Device
}

var _ model.UnderlyingNetwork = &GvisorStack{}

// GvisorMTU is the MTU used by [NewGvisorStack]. We use this specific MTU value
// because it allows lucas-clemente/quic-go to work as intended.
const GvisorMTU = 1300

// NewGvisorStack constructs a new [GvisorStack] instance. This function calls
// [runtimex.PanicOnError] in case of failure.
//
// Arguments:
//
// - laddr is the IPv4 or IPv6 address to assign to the [GvisorStack];
//
// - cfg contains TLS MITM configuration;
//
// - ggi provides the getaddrinfo functionality to the [GvisorStack].
func NewGvisorStack(laddr string, cfg *TLSMITMConfig, ggi GvisorGetaddrinfo) *GvisorStack {
	// parse the local address
	addr := runtimex.Try1(netip.ParseAddr(laddr))

	// create userspace TUN and network stack
	tun, net := runtimex.Try2(netstack.CreateNetTUN([]netip.Addr{addr}, []netip.Addr{}, GvisorMTU))

	// fill and return the network
	return &GvisorStack{
		gginfo:         ggi,
		ipAddress:      addr,
		netStack:       net,
		tlsMITMmConfig: cfg,
		tunDevice:      tun,
	}
}

// IPAddress returns the IP address assigned to the stack.
func (gs *GvisorStack) IPAddress() string {
	return gs.ipAddress.String()
}

// Attach starts two background goroutines that will run until
// the given context has not been canceled. The first goroutine will
// read incoming packets from the given NIC and write them to the
// network stack. The second goroutine will read packets generated
// by the network stack and write packets to the NIC.
func (gs *GvisorStack) Attach(ctx context.Context, nic *NIC) {
	log.Infof("netem: ifconfig %s %s", nic.name, gs.ipAddress)

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go gvisorReadFromTUN(ctx, wg, gs.tunDevice, nic)

	wg.Add(1)
	go gvisorWriteToTUN(ctx, wg, gs.tunDevice, nic)

	go func() {
		wg.Wait()
		log.Infof("netem: ifconfig %s down", nic.name)
	}()
}

// gvisorReadFromTUN reads outgoing packets from the given TUN device
// and writes them to the outgoing channel of the NIC.
func gvisorReadFromTUN(ctx context.Context, wg *sync.WaitGroup, tun tun.Device, nic *NIC) {
	defer wg.Done()
	for {
		// create buffer for incoming packet
		const packetbuffer = 1 << 17
		runtimex.Assert(packetbuffer > GvisorMTU, "packetbuffer smaller than the MTU")
		buffer := make([]byte, packetbuffer)

		// read incoming packet
		count, err := tun.Read(buffer, 0)
		if err != nil {
			log.Warnf("netem: gvisorReadFromTUN: %s", err.Error())
			return
		}
		rawPacket := buffer[:count]

		// write incoming packet to the NIC
		if err := nic.WriteOutgoing(ctx, rawPacket); err != nil {
			if !errors.Is(err, ErrNICBufferFull) {
				log.Warnf("netem: gvisorReadFromTUN: %s", err.Error())
				return
			}
		}
	}
}

// gvisorWriteToTUN reads incoming packets from the NIC incoming
// channel and writes them to the given TUN device.
func gvisorWriteToTUN(ctx context.Context, wg *sync.WaitGroup, tun tun.Device, nic *NIC) {
	defer wg.Done()
	for {
		// read incoming packet
		rawPacket, err := nic.ReadIncoming(ctx)
		if err != nil {
			log.Warnf("netem: gvisorWriteToTun: %s", err.Error())
			return
		}

		// write packet to the tun device
		if _, err := tun.Write(rawPacket, 0); err != nil {
			log.Warnf("netem: gvisorWriteToTun: %s", err.Error())
			return
		}
	}
}

// DefaultCertPool implements model.UnderlyingNetwork.
func (gs *GvisorStack) DefaultCertPool() *x509.CertPool {
	return gs.tlsMITMmConfig.CertPool()
}

// DialContext implements model.UnderlyingNetwork.
func (gs *GvisorStack) DialContext(
	ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
	// if needed, configure the timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	// dial the actual connection
	conn, err := gs.netStack.DialContext(ctx, network, address)
	if err != nil {
		return nil, mapGvisorError(err)
	}

	// wrap returned connection to correctly map errors
	return &gvisorConnWrapper{conn}, nil
}

// GetaddrinfoLookupANY implements model.UnderlyingNetwork.
func (gs *GvisorStack) GetaddrinfoLookupANY(
	ctx context.Context, domain string) ([]string, string, error) {
	return gs.gginfo.Lookup(ctx, domain)
}

// GetaddrinfoResolverNetwork implements model.UnderlyingNetwork
func (gs *GvisorStack) GetaddrinfoResolverNetwork() string {
	return "getaddrinfo" // pretend we are calling the getaddrinfo(3) func
}

// ListenUDP implements model.UnderlyingNetwork.
func (gs *GvisorStack) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	runtimex.Assert(network == "udp", "expected network to be 'udp'")
	pconn, err := gs.netStack.ListenUDP(addr)
	if err != nil {
		return nil, mapGvisorError(err)
	}
	return &gvisorPacketConnWrapper{pconn}, nil
}

// Listen returns a listening TCP connection or an error.
func (gs *GvisorStack) ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error) {
	runtimex.Assert(network == "tcp", "expected network to be 'tcp'")
	listener, err := gs.netStack.ListenTCP(addr)
	if err != nil {
		return nil, mapGvisorError(err)
	}
	return &gvisorListenerWrapper{listener}, nil
}

// gvisorSuffixToError maps a suffix to an stdlib error.
type gvisorSuffixToError struct {
	// suffix is the gvisor err.Error() suffix.
	suffix string

	// err is generally a syscall error but it could
	// also be any other stdlib error.
	err error
}

// allGvisorSyscallErrors defines [gvisorSuffixToError] rules for all the
// errors emitted by gvisor that matter to measuring censorship.
//
// See https://github.com/google/gvisor/blob/master/pkg/tcpip/errors.go
var allGvisorSyscallErrors = []*gvisorSuffixToError{{
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

// mapGvisorError maps a gvisor error to an stdlib error.
func mapGvisorError(err error) error {
	if err != nil {
		estring := err.Error()
		for _, entry := range allGvisorSyscallErrors {
			if strings.HasSuffix(estring, entry.suffix) {
				return entry.err
			}
		}
	}
	return err
}

// gvisorConnWrapper wraps a [net.Conn] to remap gvisor errors
// so that we can emulate stdlib errors.
type gvisorConnWrapper struct {
	c net.Conn
}

var _ net.Conn = &gvisorConnWrapper{}

// Close implements net.Conn
func (gcw *gvisorConnWrapper) Close() error {
	return gcw.c.Close()
}

// LocalAddr implements net.Conn
func (gcw *gvisorConnWrapper) LocalAddr() net.Addr {
	return gcw.c.LocalAddr()
}

// Read implements net.Conn
func (gcw *gvisorConnWrapper) Read(b []byte) (n int, err error) {
	count, err := gcw.c.Read(b)
	return count, mapGvisorError(err)
}

// RemoteAddr implements net.Conn
func (gcw *gvisorConnWrapper) RemoteAddr() net.Addr {
	return gcw.c.RemoteAddr()
}

// SetDeadline implements net.Conn
func (gcw *gvisorConnWrapper) SetDeadline(t time.Time) error {
	return gcw.c.SetDeadline(t)
}

// SetReadDeadline implements net.Conn
func (gcw *gvisorConnWrapper) SetReadDeadline(t time.Time) error {
	return gcw.c.SetReadDeadline(t)
}

// SetWriteDeadline implements net.Conn
func (gcw *gvisorConnWrapper) SetWriteDeadline(t time.Time) error {
	return gcw.c.SetWriteDeadline(t)
}

// Write implements net.Conn
func (gcw *gvisorConnWrapper) Write(b []byte) (n int, err error) {
	count, err := gcw.c.Write(b)
	return count, mapGvisorError(err)
}

// gvisorPacketConnWrapper wraps a [model.UDPLikeConn] such that we can use
// this connection with lucas-clemente/quic-go and remaps gvisor errors to
// emulate actual stdlib errors.
type gvisorPacketConnWrapper struct {
	c *gonet.UDPConn
}

var (
	_ model.UDPLikeConn = &gvisorPacketConnWrapper{}
	_ syscall.RawConn   = &gvisorPacketConnWrapper{}
)

// Close implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) Close() error {
	return gpcw.c.Close()
}

// LocalAddr implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) LocalAddr() net.Addr {
	return gpcw.c.LocalAddr()
}

// ReadFrom implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) ReadFrom(p []byte) (int, net.Addr, error) {
	count, addr, err := gpcw.c.ReadFrom(p)
	return count, addr, mapGvisorError(err)
}

// SetDeadline implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) SetDeadline(t time.Time) error {
	return gpcw.c.SetDeadline(t)
}

// SetReadDeadline implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) SetReadDeadline(t time.Time) error {
	return gpcw.c.SetReadDeadline(t)
}

// SetWriteDeadline implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) SetWriteDeadline(t time.Time) error {
	return gpcw.c.SetWriteDeadline(t)
}

// WriteTo implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) WriteTo(p []byte, addr net.Addr) (int, error) {
	count, err := gpcw.c.WriteTo(p, addr)
	return count, mapGvisorError(err)
}

// Implementation note: the following function calls are all stubs and they
// should nonetheless work with lucas-clemente/quic-go.

// SetReadBuffer implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) SetReadBuffer(bytes int) error {
	log.Infof("netem: SetReadBuffer stub called with %d bytes as the argument", bytes)
	return nil
}

// SyscallConn implements model.UDPLikeConn
func (gpcw *gvisorPacketConnWrapper) SyscallConn() (syscall.RawConn, error) {
	log.Infof("netem: SyscallConn stub called")
	return gpcw, nil
}

// Control implements syscall.RawConn
func (gpcw *gvisorPacketConnWrapper) Control(f func(fd uintptr)) error {
	log.Infof("netem: Control stub called")
	return nil
}

// Read implements syscall.RawConn
func (gpcw *gvisorPacketConnWrapper) Read(f func(fd uintptr) (done bool)) error {
	log.Infof("netem: Read stub called")
	return nil
}

// Write implements syscall.RawConn
func (gpcw *gvisorPacketConnWrapper) Write(f func(fd uintptr) (done bool)) error {
	log.Infof("netem: Write stub called")
	return nil
}

// gvisorListenerWrapper wraps a [net.Listener] and maps gvisor
// errors to the corresponding stdlib errors.
type gvisorListenerWrapper struct {
	l *gonet.TCPListener
}

var _ net.Listener = &gvisorListenerWrapper{}

// Accept implements net.Listener
func (glw *gvisorListenerWrapper) Accept() (net.Conn, error) {
	conn, err := glw.l.Accept()
	if err != nil {
		return nil, mapGvisorError(err)
	}
	return &gvisorConnWrapper{conn}, nil
}

// Addr implements net.Listener
func (glw *gvisorListenerWrapper) Addr() net.Addr {
	return glw.l.Addr()
}

// Close implements net.Listener
func (glw *gvisorListenerWrapper) Close() error {
	return glw.l.Close()
}
