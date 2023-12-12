package testingsocks5

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"

	"context"
)

const (
	connectCommand   = uint8(1)
	bindCommand      = uint8(2)
	associateCommand = uint8(3)
	ipv4Address      = uint8(1)
	fqdnAddress      = uint8(3)
	ipv6Address      = uint8(4)
)

const (
	successReply uint8 = iota
	serverFailure
	ruleFailure
	networkUnreachable
	hostUnreachable
	connectionRefused
	ttlExpired
	commandNotSupported
	addrTypeNotSupported
)

var (
	errUnrecognizedAddrType = fmt.Errorf("unrecognized address type")
)

// addrSpec is used to return the target addrSpec
// which may be specified as IPv4, IPv6, or a FQDN
type addrSpec struct {
	Address string
	Port    int
}

// A request represents request received by a server
type request struct {
	// Protocol version
	Version uint8

	// Requested command
	Command uint8

	// AddrSpec of the desired destination
	DestAddr *addrSpec
}

// newRequest creates a new request from the tcp connection
func newRequest(cconn net.Conn) (*request, error) {
	// Read the version byte
	header := []byte{0, 0, 0}
	if _, err := io.ReadFull(cconn, header); err != nil {
		return nil, fmt.Errorf("failed to get command version: %w", err)
	}

	// Ensure we are compatible
	if header[0] != socks5Version {
		return nil, fmt.Errorf("unsupported command version: %v", header[0])
	}

	// Read in the destination address
	dest, err := readAddrSpec(cconn)
	if err != nil {
		return nil, err
	}

	request := &request{
		Version:  socks5Version,
		Command:  header[1],
		DestAddr: dest,
	}

	return request, nil
}

// handleRequest is used for request processing after authentication
func (s *Server) handleRequest(req *request, cconn net.Conn) error {
	ctx := context.Background()
	if req.Command != connectCommand {
		return sendReply(cconn, commandNotSupported, &net.TCPAddr{})
	}
	return s.handleConnect(ctx, cconn, req)
}

// handleConnect is used to handle a connect command
func (s *Server) handleConnect(ctx context.Context, cconn net.Conn, req *request) error {
	s.logger.Info("handling CONNECT command")

	// Attempt to connect
	endpoint := net.JoinHostPort(req.DestAddr.Address, strconv.Itoa(req.DestAddr.Port))
	s.logger.Infof("endpoint: %s", endpoint)
	dialer := s.netx.NewDialerWithResolver(s.logger, s.netx.NewStdlibResolver(s.logger))
	sconn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		// Note: the original go-socks5 selects the proper error but it does not
		// matter for our purposes, so we always return hostUnreachable.
		return sendReply(cconn, hostUnreachable, &net.TCPAddr{})
	}
	defer sconn.Close()

	// Send success
	local := sconn.LocalAddr().(*net.TCPAddr)
	if err := sendReply(cconn, successReply, local); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}

	// Start proxying
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go func() {
		_, _ = io.Copy(cconn, sconn)
		wg.Done()
	}()
	go func() {
		_, _ = io.Copy(sconn, cconn)
		wg.Done()
	}()
	wg.Wait()
	return nil
}

// readAddrSpec is used to read AddrSpec.
// Expects an address type byte, follwed by the address and port.
func readAddrSpec(cconn net.Conn) (*addrSpec, error) {
	d := &addrSpec{}

	// Get the address type
	addrType := []byte{0}
	if _, err := io.ReadFull(cconn, addrType); err != nil {
		return nil, err
	}

	// Handle on a per type basis
	switch addrType[0] {
	case ipv4Address:
		addr := make([]byte, 4)
		if _, err := io.ReadFull(cconn, addr); err != nil {
			return nil, err
		}
		d.Address = net.IP(addr).String()

	case ipv6Address:
		addr := make([]byte, 16)
		if _, err := io.ReadFull(cconn, addr); err != nil {
			return nil, err
		}
		d.Address = net.IP(addr).String()

	case fqdnAddress:
		lengthBuffer := []byte{0}
		if _, err := io.ReadFull(cconn, lengthBuffer); err != nil {
			return nil, err
		}
		addrLen := int(lengthBuffer[0])
		fqdn := make([]byte, addrLen)
		if _, err := io.ReadFull(cconn, fqdn); err != nil {
			return nil, err
		}
		d.Address = string(fqdn)

	default:
		return nil, errUnrecognizedAddrType
	}

	// Read the port
	port := []byte{0, 0}
	if _, err := io.ReadFull(cconn, port); err != nil {
		return nil, err
	}
	d.Port = (int(port[0]) << 8) | int(port[1])

	return d, nil
}

// sendReply is used to send a reply message
func sendReply(w io.Writer, resp uint8, addr *net.TCPAddr) error {
	// Format the address
	var (
		addrType uint8
		addrBody []byte
		addrPort uint16
	)

	// Note: the order of these cases matters!
	switch {
	case addr.IP.To4() != nil:
		addrType = ipv4Address
		addrBody = []byte(addr.IP.To4())
		addrPort = uint16(addr.Port)

	case addr.IP.To16() != nil:
		addrType = ipv6Address
		addrBody = []byte(addr.IP.To16())
		addrPort = uint16(addr.Port)

	default:
		addrType = ipv4Address
		addrBody = []byte{0, 0, 0, 0}
		addrPort = 0
	}

	// Format the message
	msg := make([]byte, 6+len(addrBody))
	msg[0] = socks5Version
	msg[1] = resp
	msg[2] = 0 // Reserved
	msg[3] = addrType
	copy(msg[4:], addrBody)
	msg[4+len(addrBody)] = byte(addrPort >> 8)
	msg[4+len(addrBody)+1] = byte(addrPort & 0xff)

	// Send the message
	_, err := w.Write(msg)
	return err
}
