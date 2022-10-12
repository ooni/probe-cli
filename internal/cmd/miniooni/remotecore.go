package main

//
// Core remote implementation
//

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/google/shlex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/songgao/water"
	"golang.org/x/sys/execabs"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

const (
	// remoteTUNDeviceName is the name assigned to the TUN device by the server.
	remoteTUNDeviceName = "miniooni0"

	// remoteServerAddr is the address assigned to the server.
	remoteServerAddr = "10.14.17.1"
)

var (
	// remoteClientAddr is the address assigned to the client.
	remoteClientAddr = netip.MustParseAddr("10.14.17.4")

	// remoteResolvers are the IP addresses used to implement getaddrinfo on the remote.
	remoteResolvers = []netip.Addr{
		netip.MustParseAddr("8.8.8.8"),
		netip.MustParseAddr("8.8.4.4"),
	}
)

// remoteServerConfig contains server configuration for remote operations.
type remoteServerConfig struct {
	// iface is the output interface to use.
	iface string
}

// remoteServerListenerFactory creates a remoteServerListener.
type remoteServerListenerFactory interface {
	// Listen returns a new listener instance or an error.
	Listen() (remoteServerListener, error)
}

// remoteServerListener creates remoteConns.
type remoteServerListener interface {
	// Accept should return a new remoteConn or an error. This function
	// MUST return net.ErrClosed after Close has been called.
	Accept() (remoteConn, error)

	// Close closes the listener.
	Close() error
}

// remoteConn is a connection between a server and a remote miniooni client.
type remoteConn interface {
	io.Reader
	io.Writer
	io.Closer
}

// remoteServerMain is the main of a remote subcommand.
func remoteServerMain(config *remoteServerConfig, factory remoteServerListenerFactory) error {
	// create the listener
	listener, err := factory.Listen()
	if err != nil {
		return err
	}
	defer listener.Close()

	// create the TUN device
	tunConfig := water.Config{
		DeviceType: water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{
			Name: remoteTUNDeviceName,
		},
	}
	tun, err := water.New(tunConfig)
	if err != nil {
		log.Errorf("remote: water.New failed: %s", err.Error())
		return err
	}
	defer tun.Close()

	// assign the correct IP address to the TUN device
	if err := remoteServerAssignAddress(config); err != nil {
		log.Errorf("remote: cannot assign address to TUN device: %s", err.Error())
		return err
	}
	defer remoteServerCleanupIPTables(config)

	// listen for signals and cleanup when we receive them
	sigch := make(chan os.Signal, 1)
	signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigch
		log.Infof("remote: interrupted by signal")
		listener.Close()
	}()

	// accept incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return nil // this is how we terminate successfully
		}
		if err != nil {
			log.Warnf("remote: listener.Accept failed: %s", err.Error())
			continue
		}

		// route traffic
		go remoteServerRoute(conn, tun)
	}
}

// remoteServerRoute routes traffic between the remote conn and the TUN device.
func remoteServerRoute(conn remoteConn, tun *water.Interface) {
	// route from the remote conn to the TUN device
	go func() {
		for {
			pkt, err := remoteReadPacket(conn)
			if err != nil {
				log.Warnf("remote: cannot read from conn: %s", err.Error())
				return
			}
			if _, err := tun.Write(pkt); err != nil {
				log.Warnf("remote: cannot write to TUN device: %s", err.Error())
				return
			}
		}
	}()

	// route from the TUN device to the remote conn
	go func() {
		buffer := make([]byte, remoteMaxPacketSize)
		for {
			count, err := tun.Read(buffer)
			if err != nil {
				log.Warnf("remote: cannot read from TUN device: %s", err.Error())
				return
			}
			pkt := buffer[:count]
			if err := remoteWritePacket(conn, pkt); err != nil {
				log.Warnf("remote: cannot write to conn: %s", err.Error())
				return
			}
		}
	}()
}

// remoteReadPacket reads a packet from conn.
func remoteReadPacket(conn io.Reader) ([]byte, error) {
	header := make([]byte, 3)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	var length int
	length |= int(header[0]) << 16
	length |= int(header[1]) << 8
	length |= int(header[2]) << 0
	pkt := make([]byte, length)
	if _, err := io.ReadFull(conn, pkt); err != nil {
		return nil, err
	}
	return pkt, nil
}

// remoteMaxPacketSize is the maximum packet size.
const remoteMaxPacketSize = (1 << 24) - 1

// errRemotePacketTooBig indicates that a packet is too big
var errRemotePacketTooBig = errors.New("packet too big")

// remoteWritePacket writes a packet to the conn.
func remoteWritePacket(conn io.Writer, pkt []byte) error {
	length := len(pkt)
	if length > remoteMaxPacketSize {
		return errRemotePacketTooBig
	}
	data := make([]byte, 3)
	data[0] = byte((length >> 16) & 0xff)
	data[1] = byte((length >> 8) & 0xff)
	data[2] = byte((length >> 0) & 0xff)
	data = append(data, pkt...)
	_, err := conn.Write(data)
	return err
}

// remoteServerAssignAddress assigns an address to the TUN device.
func remoteServerAssignAddress(config *remoteServerConfig) error {
	script := []string{
		fmt.Sprintf("ip addr add %s/24 dev %s", remoteServerAddr, remoteTUNDeviceName),
		fmt.Sprintf("ip link set dev %s up", remoteTUNDeviceName),
		fmt.Sprintf("iptables -t nat -I POSTROUTING -o %s -j MASQUERADE", config.iface),
		"sysctl net.ipv4.ip_forward=1",
	}
	for _, cmd := range script {
		if err := remoteServerExec(cmd); err != nil {
			return err
		}
	}
	return nil
}

// remoteServerExec executes a command.
func remoteServerExec(cmdline string) error {
	argv, err := shlex.Split(cmdline)
	runtimex.PanicOnError(err, "shlex.Split failed")
	runtimex.Assert(len(argv) >= 1, "expected at least one argv entry")
	cmd := execabs.Command(argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	log.Infof("remote: exec: %s", cmd.String())
	return cmd.Run()
}

// remoteServerCleanupIPTables removes iptables rules we have added.
func remoteServerCleanupIPTables(config *remoteServerConfig) {
	remoteServerExec(fmt.Sprintf(
		"iptables -t nat -D POSTROUTING -o %s -j MASQUERADE",
		config.iface,
	))
}

// remoteClientDialer creates connections.
type remoteClientDialer interface {
	Dial() (remoteConn, error)
}

// remoteClient is a client for the remote protocol
type remoteClient struct {
	// closeOnce allows to call Close just once
	closeOnce *sync.Once

	// conn is the transport connection.
	conn remoteConn

	// net is the underlying userspace network stack
	net *netstack.Net

	// tun is the TUN device in userspace.
	tun tun.Device
}

// newRemoteClient creates a new remote client.
func newRemoteClient(dialer remoteClientDialer) (*remoteClient, error) {
	// establish a connection with the remote host
	conn, err := dialer.Dial()
	if err != nil {
		return nil, err
	}

	const mtu = 1300 // must be >= 1252, which is used by quic-go

	// create the TUN device in userspace
	tun, net, err := netstack.CreateNetTUN(
		[]netip.Addr{remoteClientAddr},
		remoteResolvers,
		mtu,
	)
	if err != nil {
		conn.Close()
		return nil, err
	}

	client := &remoteClient{
		closeOnce: &sync.Once{},
		net:       net,
		tun:       tun,
		conn:      conn,
	}
	return client, nil
}

// Close closes the connections used by a client.
func (c *remoteClient) Close() error {
	var err error
	c.closeOnce.Do(func() {
		if e := c.tun.Close(); e != nil {
			err = e
		}
		if e := c.conn.Close(); e != nil && err == nil {
			err = e
		}
	})
	return err
}

// route routes the traffic
func (c *remoteClient) route() {
	// the following code has been adapted from ooni/minivpn
	const zeroOffset = 0

	go func() {
		for {
			pkt, err := remoteReadPacket(c.conn)
			if err != nil {
				log.Errorf("remote: cannot read from conn: %s", err.Error())
				return
			}
			if _, err = c.tun.Write(pkt, zeroOffset); err != nil {
				log.Errorf("remote: cannot write to TUN device: %v", err)
				break
			}
		}
	}()

	go func() {
		buf := make([]byte, remoteMaxPacketSize)
		for {
			count, err := c.tun.Read(buf, zeroOffset)
			if err != nil {
				log.Errorf("remote: cannot read from TUN device: %v", err)
				break
			}
			pkt := buf[:count]
			if err := remoteWritePacket(c.conn, pkt); err != nil {
				log.Errorf("remote: cannot write to conn: %s", err.Error())
				return
			}
		}
	}()
}

// DialContext implements UnderlyingNetwork.DialContext.
func (c *remoteClient) DialContext(ctx context.Context, timeout time.Duration, network string, address string) (net.Conn, error) {
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if remoteIsIPv6(address) {
		// TODO(bassosimone): extend this implementation to support IPv6
		return nil, syscall.EHOSTUNREACH
	}
	return c.net.DialContext(ctx, network, address)
}

// remoteIsIPv6 returns whether the given endpoint contains an IPv6 address
func remoteIsIPv6(endpoint string) bool {
	addr, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return false
	}
	v6, err := netxlite.IsIPv6(addr)
	if err != nil {
		return false
	}
	return v6
}

// ListenUDP implements UnderlyingNetwork.
func (c *remoteClient) ListenUDP(network string, addr *net.UDPAddr) (model.UDPLikeConn, error) {
	pconn, err := c.net.ListenUDP(addr)
	if err != nil {
		return nil, err
	}
	pwrap := &remoteClientUDPConn{pconn}
	return pwrap, nil
}

// remoteClientUDPConn adapts to model.UDPLikeConn.
type remoteClientUDPConn struct {
	net.PacketConn
}

// WriteTo implements net.PacketConn.
func (c *remoteClientUDPConn) WriteTo(pkt []byte, dest net.Addr) (int, error) {
	if remoteIsIPv6(dest.String()) {
		// TODO(bassosimone): extend this implementation to support IPv6
		return 0, syscall.EHOSTUNREACH
	}
	return c.PacketConn.WriteTo(pkt, dest)
}

// SetReadBuffer allows setting the read buffer.
func (c *remoteClientUDPConn) SetReadBuffer(bytes int) error {
	return nil
}

// SyscallConn returns a conn suitable for calling syscalls,
// which is also instrumental to setting the read buffer.
//
// We need to mock SyscallConn and return a fake syscall.RawConn
// because otherwise lucas-clemente/quic-go would not work as intended.
func (c *remoteClientUDPConn) SyscallConn() (syscall.RawConn, error) {
	return &remoteClientRawConnUDP{}, nil
}

// remoteClientRawConnUDP implements syscall.RawConn
type remoteClientRawConnUDP struct{}

// Control implements syscall.RawConn
func (*remoteClientRawConnUDP) Control(f func(fd uintptr)) error {
	return nil
}

// Read implements syscall.RawConn
func (*remoteClientRawConnUDP) Read(f func(fd uintptr) (done bool)) error {
	return nil
}

// Write implements syscall.RawConn
func (*remoteClientRawConnUDP) Write(f func(fd uintptr) (done bool)) error {
	return nil
}

// GetaddrinfoLookupANY implements UnderlyingNetwork.
func (c *remoteClient) GetaddrinfoLookupANY(ctx context.Context, domain string) ([]string, string, error) {
	addrs, err := c.net.LookupContextHost(ctx, domain)
	return addrs, "", err
}

// GetaddrinfoResolverNetwork implements UnderlyingNetwork.
func (c *remoteClient) GetaddrinfoResolverNetwork() string {
	return netxlite.StdlibResolverGetaddrinfo
}
