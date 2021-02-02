package resolver

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"time"
)

// DialContextFunc is a generic function for dialing a connection.
type DialContextFunc func(context.Context, string, string) (net.Conn, error)

// DNSOverTCP is a DNS over TCP/TLS RoundTripper. Use NewDNSOverTCP
// and NewDNSOverTLS to create specific instances that use plaintext
// queries or encrypted queries over TLS.
//
// As a known bug, this implementation always creates a new connection
// for each incoming query, thus increasing the response delay.
type DNSOverTCP struct {
	dial            DialContextFunc
	address         string
	network         string
	requiresPadding bool
}

// NewDNSOverTCP creates a new DNSOverTCP transport.
func NewDNSOverTCP(dial DialContextFunc, address string) DNSOverTCP {
	return DNSOverTCP{
		dial:            dial,
		address:         address,
		network:         "tcp",
		requiresPadding: false,
	}
}

// NewDNSOverTLS creates a new DNSOverTLS transport.
func NewDNSOverTLS(dial DialContextFunc, address string) DNSOverTCP {
	return DNSOverTCP{
		dial:            dial,
		address:         address,
		network:         "dot",
		requiresPadding: true,
	}
}

// RoundTrip implements RoundTripper.RoundTrip.
func (t DNSOverTCP) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	if len(query) > math.MaxUint16 {
		return nil, errors.New("query too long")
	}
	conn, err := t.dial(ctx, "tcp", t.address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return nil, err
	}
	// Write request
	buf := []byte{byte(len(query) >> 8)}
	buf = append(buf, byte(len(query)))
	buf = append(buf, query...)
	if _, err = conn.Write(buf); err != nil {
		return nil, err
	}
	// Read response
	header := make([]byte, 2)
	if _, err = io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(header[0])<<8 | int(header[1])
	reply := make([]byte, length)
	if _, err = io.ReadFull(conn, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// RequiresPadding returns true for DoT and false for TCP
// according to RFC8467.
func (t DNSOverTCP) RequiresPadding() bool {
	return t.requiresPadding
}

// Network returns the transport network (e.g., doh, dot)
func (t DNSOverTCP) Network() string {
	return t.network
}

// Address returns the upstream server address.
func (t DNSOverTCP) Address() string {
	return t.address
}

var _ RoundTripper = DNSOverTCP{}
