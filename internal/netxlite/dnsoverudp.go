package netxlite

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSOverUDPTransport is a DNS-over-UDP DNSTransport.
type DNSOverUDPTransport struct {
	dialer  model.Dialer
	address string
}

// NewDNSOverUDPTransport creates a DNSOverUDPTransport instance.
//
// Arguments:
//
// - dialer is any type that implements the Dialer interface;
//
// - address is the endpoint address (e.g., 8.8.8.8:53).
func NewDNSOverUDPTransport(dialer model.Dialer, address string) *DNSOverUDPTransport {
	return &DNSOverUDPTransport{dialer: dialer, address: address}
}

// RoundTrip sends a query and receives a reply.
func (t *DNSOverUDPTransport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	conn, err := t.dialer.DialContext(ctx, "udp", t.address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	// Use five seconds timeout like Bionic does. See
	// https://labs.ripe.net/Members/baptiste_jonglez_1/persistent-dns-connections-for-reliability-and-performance
	if err = conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}
	if _, err = conn.Write(query); err != nil {
		return nil, err
	}
	reply := make([]byte, 1<<17)
	var n int
	n, err = conn.Read(reply)
	if err != nil {
		return nil, err
	}
	return reply[:n], nil
}

// RequiresPadding returns false for UDP according to RFC8467.
func (t *DNSOverUDPTransport) RequiresPadding() bool {
	return false
}

// Network returns the transport network, i.e., "udp".
func (t *DNSOverUDPTransport) Network() string {
	return "udp"
}

// Address returns the upstream server address.
func (t *DNSOverUDPTransport) Address() string {
	return t.address
}

// CloseIdleConnections closes idle connections, if any.
func (t *DNSOverUDPTransport) CloseIdleConnections() {
	// nothing to do
}

var _ model.DNSTransport = &DNSOverUDPTransport{}
