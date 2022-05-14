package netxlite

import (
	"context"
	"errors"
	"io"
	"math"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DialContextFunc is the type of net.Dialer.DialContext.
type DialContextFunc func(context.Context, string, string) (net.Conn, error)

// DNSOverTCPTransport is a DNS-over-{TCP,TLS} DNSTransport.
//
// Bug: this implementation always creates a new connection for each query.
type DNSOverTCPTransport struct {
	dial            DialContextFunc
	address         string
	network         string
	requiresPadding bool
}

// NewDNSOverTCPTransport creates a new DNSOverTCPTransport.
//
// Arguments:
//
// - dial is a function with the net.Dialer.DialContext's signature;
//
// - address is the endpoint address (e.g., 8.8.8.8:53).
func NewDNSOverTCPTransport(dial DialContextFunc, address string) *DNSOverTCPTransport {
	return &DNSOverTCPTransport{
		dial:            dial,
		address:         address,
		network:         "tcp",
		requiresPadding: false,
	}
}

// NewDNSOverTLS creates a new DNSOverTLS transport.
//
// Arguments:
//
// - dial is a function with the net.Dialer.DialContext's signature;
//
// - address is the endpoint address (e.g., 8.8.8.8:853).
func NewDNSOverTLS(dial DialContextFunc, address string) *DNSOverTCPTransport {
	return &DNSOverTCPTransport{
		dial:            dial,
		address:         address,
		network:         "dot",
		requiresPadding: true,
	}
}

// RoundTrip sends a query and receives a reply.
func (t *DNSOverTCPTransport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
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
func (t *DNSOverTCPTransport) RequiresPadding() bool {
	return t.requiresPadding
}

// Network returns the transport network, i.e., "dot" or "tcp".
func (t *DNSOverTCPTransport) Network() string {
	return t.network
}

// Address returns the upstream server endpoint (e.g., "1.1.1.1:853").
func (t *DNSOverTCPTransport) Address() string {
	return t.address
}

// CloseIdleConnections closes idle connections, if any.
func (t *DNSOverTCPTransport) CloseIdleConnections() {
	// nothing to do
}

var _ model.DNSTransport = &DNSOverTCPTransport{}
