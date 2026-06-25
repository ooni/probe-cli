package tlsmiddlebox

//
// Custom TTL dialer
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.org/x/sys/unix"
)

const timeout time.Duration = 15 * time.Second

// Define the Linux socket option enabling error queueing
const IP_RECVERR = 12

// Define the Linux recvmsg flag for reading the error queue
const MSG_ERRQUEUE = 2

func NewDialerTTLWrapper(localPort int) model.Dialer {
	return &dialerTTLWrapper{
		Dialer: &net.Dialer{
			Timeout: timeout,
			LocalAddr: &net.TCPAddr{
				Port: localPort,
			},
		},
	}
}

// dialerTTLWrapper wraps errors and also returns a TTL wrapped conn
type dialerTTLWrapper struct {
	Dialer model.SimpleDialer
}

// ttlConn wraps the TCP connection
type ttlConn struct {
	*net.TCPConn
	fd int
}

type Hop struct {
	Addr net.IP
	Type uint8
	Code uint8
}

var _ model.Dialer = &dialerTTLWrapper{}

// DialContext implements model.Dialer.DialContext
func (d *dialerTTLWrapper) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	tcp := conn.(*net.TCPConn)

	raw, err := tcp.SyscallConn()
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	// Extract underlying socket file descriptor from TCP connection
	var fd int
	err = raw.Control(func(f uintptr) {
		fd = int(f)
	})

	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	// Set the IP_RECVERR socket option to enable ICMP errors to be stored in the socket error queue
	// Such errors will be subsequently read with MSG_ERRQUEUE
	err = raw.Control(func(f uintptr) {
		unix.SetsockoptInt(int(f), unix.IPPROTO_IP, IP_RECVERR, 1)
	})

	// Set the SO_TIMESTAMPNS socket option to enable

	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	return &ttlConn{
		TCPConn: tcp,
		fd:      fd,
	}, nil
}

// CloseIdleConnections implements model.Dialer.CloseIdleConnections
func (d *dialerTTLWrapper) CloseIdleConnections() {
	// nothing to do here
}

// ReadErrQueue() uses MSG_ERRQUEUE to read ICMP messages
func (c *ttlConn) ReadErrQueue() (*Hop, error) {
	buf := make([]byte, 256)
	oob := make([]byte, 512)

	_, oobn, _, _, err := unix.Recvmsg(
		c.fd,
		buf,
		oob,
		MSG_ERRQUEUE,
	)

	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	cms, err := unix.ParseSocketControlMessage((oob[:oobn]))
	if err != nil {
		return nil, err
	}

	for _, cm := range cms {
		if cm.Header.Level != unix.IPPROTO_IP {
			continue
		}

		if cm.Header.Type != IP_RECVERR {
			continue
		}

		ee := cm.Data

		if len(ee) < 16 {
			continue
		}

		icmpType := ee[4]
		icmpCode := ee[5]

		sa := ee[16:]

		if len(sa) < 8 {
			continue
		}

		routerIP := net.IPv4(sa[4], sa[5], sa[6], sa[7])

		return &Hop{
			Addr: routerIP,
			Type: icmpType,
			Code: icmpCode,
		}, nil

	}

	return nil, nil
}
