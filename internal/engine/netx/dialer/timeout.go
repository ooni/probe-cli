package dialer

import (
	"context"
	"net"
	"time"
)

// TimeoutDialer is a Dialer that enforces a timeout
type TimeoutDialer struct {
	Dialer
	ConnectTimeout time.Duration // default: 30 seconds
}

// DialContext implements Dialer.DialContext
func (d TimeoutDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	timeout := 30 * time.Second
	if d.ConnectTimeout != 0 {
		timeout = d.ConnectTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return d.Dialer.DialContext(ctx, network, address)
}
