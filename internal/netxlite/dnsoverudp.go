package netxlite

import (
	"context"
	"time"
)

// DNSOverUDP is a DNS-over-UDP DNSTransport.
type DNSOverUDP struct {
	dialer  Dialer
	address string
}

// NewDNSOverUDP creates a DNSOverUDP instance.
//
// Arguments:
//
// - dialer is any type that implements the Dialer interface;
//
// - address is the endpoint address (e.g., 8.8.8.8:53).
func NewDNSOverUDP(dialer Dialer, address string) *DNSOverUDP {
	return &DNSOverUDP{dialer: dialer, address: address}
}

// RoundTrip sends a query and receives a reply.
func (t *DNSOverUDP) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
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
func (t *DNSOverUDP) RequiresPadding() bool {
	return false
}

// Network returns the transport network, i.e., "udp".
func (t *DNSOverUDP) Network() string {
	return "udp"
}

// Address returns the upstream server address.
func (t *DNSOverUDP) Address() string {
	return t.address
}

// CloseIdleConnections closes idle connections, if any.
func (t *DNSOverUDP) CloseIdleConnections() {
	// nothing to do
}

var _ DNSTransport = &DNSOverUDP{}
