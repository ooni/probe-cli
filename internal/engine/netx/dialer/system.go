package dialer

import (
	"context"
	"net"
	"time"
)

// underlyingDialer is the underlying net.Dialer.
var underlyingDialer = &net.Dialer{
	Timeout:   15 * time.Second,
	KeepAlive: 15 * time.Second,
}

// SystemDialer is the system dialer.
type SystemDialer struct{}

// DialContext implements Dialer.DialContext
func (d SystemDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return underlyingDialer.DialContext(ctx, network, address)
}

// Default is the dialer we use by default.
var Default = SystemDialer{}
