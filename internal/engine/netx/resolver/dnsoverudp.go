package resolver

import (
	"context"
	"net"
	"time"
)

// Dialer is the network dialer interface assumed by this package.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// DNSOverUDP is a DNS over UDP RoundTripper.
type DNSOverUDP struct {
	dialer  Dialer
	address string
}

// NewDNSOverUDP creates a DNSOverUDP instance.
func NewDNSOverUDP(dialer Dialer, address string) DNSOverUDP {
	return DNSOverUDP{dialer: dialer, address: address}
}

// RoundTrip implements RoundTripper.RoundTrip.
func (t DNSOverUDP) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
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

// RequiresPadding returns false for UDP according to RFC8467
func (t DNSOverUDP) RequiresPadding() bool {
	return false
}

// Network returns the transport network (e.g., doh, dot)
func (t DNSOverUDP) Network() string {
	return "udp"
}

// Address returns the upstream server address.
func (t DNSOverUDP) Address() string {
	return t.address
}

var _ RoundTripper = DNSOverUDP{}
