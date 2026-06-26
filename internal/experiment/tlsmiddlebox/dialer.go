package tlsmiddlebox

//
// Custom TTL dialer
//

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.org/x/sys/unix"
)

const timeout time.Duration = 15 * time.Second

// Define the Linux IPPROTO_IP socket option enabling error queueing
const IP_RECVERR = 12

// Define the Linux recvmsg flag for reading the error queue
const MSG_ERRQUEUE = 2

// Define the Linux SOL_SOCKET socket option enabling nanosecond timestamps
const SO_TIMESTAMPNS = 35

// Define the Linux socket control message type for nanosecond timestamps
const SCM_TIMESTAMPNS = 35

func NewDialerTTLWrapper() model.Dialer {
	return &dialerTTLWrapper{
		Dialer: &net.Dialer{Timeout: timeout},
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
	Addr      net.IP
	Type      uint8
	Code      uint8
	Timestamp time.Time
	TTL       uint8
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

	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	// Set the SO_TIMESTAMPNS socket option to enable nanosecond timestamps
	err = raw.Control(func(f uintptr) {
		unix.SetsockoptInt(int(f), unix.SOL_SOCKET, SO_TIMESTAMPNS, 1)
	})

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
	oob := make([]byte, 4096)

	n, oobn, _, _, err := unix.Recvmsg(
		c.fd,
		buf,
		oob,
		MSG_ERRQUEUE,
	)

	fmt.Println("recvmsg called")
	fmt.Println("n:", n, "oobn:", oobn, "err:", err)

	if err != nil {
		return nil, err
	}

	fmt.Printf("RAW OOB: %x\n", oob[:oobn])
	return nil, nil

	// buf := make([]byte, 256)
	// oob := make([]byte, 512)

	// // Read from MSG_ERRQUEUE
	// _, oobn, _, _, err := unix.Recvmsg(
	// 	c.fd,
	// 	buf,
	// 	oob,
	// 	MSG_ERRQUEUE,
	// )

	// if err != nil {
	// 	return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	// }

	// var (
	// 	hop           Hop
	// 	haveICMP      bool
	// 	haveTimestamp bool
	// 	ts            time.Time
	// )

	// cms, err := unix.ParseSocketControlMessage((oob[:oobn]))
	// if err != nil {
	// 	return nil, err
	// }

	// for i, cm := range cms {
	// 	fmt.Printf("=== CMS[%d] ===\n", i)
	// 	fmt.Printf("Level: %d Type: %d Len: %d\n",
	// 		cm.Header.Level,
	// 		cm.Header.Type,
	// 		len(cm.Data),
	// 	)

	// 	fmt.Printf("RAW DATA: %x\n", cm.Data)
	// }

	// for _, cm := range cms {

	// 	// Check for timestamp error
	// 	if cm.Header.Level == unix.SOL_SOCKET && cm.Header.Type == SCM_TIMESTAMPNS {
	// 		sec := int64(binary.LittleEndian.Uint64(cm.Data[0:8]))
	// 		nsec := int64(binary.LittleEndian.Uint64(cm.Data[8:16]))
	// 		ts = time.Unix(sec, nsec)
	// 		haveTimestamp = true
	// 		continue
	// 	}

	// 	// Check for ICMP error
	// 	if cm.Header.Level == unix.IPPROTO_IP && cm.Header.Type == IP_RECVERR {
	// 		fmt.Printf("\n--- IP_RECVERR ---\n")
	// 		fmt.Printf("len(cm.Data): %d\n", len(cm.Data))
	// 		fmt.Printf("hex dump: %x\n", cm.Data)

	// 		ee := cm.Data

	// 		if len(ee) < 16 {
	// 			continue
	// 		}

	// 		hop.Type = ee[4]
	// 		hop.Code = ee[5]

	// 		sa := ee[16:]
	// 		if len(sa) >= 8 {
	// 			hop.Addr = net.IPv4(sa[4], sa[5], sa[6], sa[7])
	// 		}

	// 		haveICMP = true
	// 	}
	// }

	// if !haveICMP {
	// 	return nil, nil
	// }

	// if haveTimestamp {
	// 	hop.Timestamp = ts
	// }

	// return &hop, nil
}
