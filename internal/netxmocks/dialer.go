package netxmocks

import (
	"context"
	"net"
)

// dialer is the interface we expect from a dialer
type dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Dialer is a mockable Dialer.
type Dialer struct {
	MockDialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

// DialContext implements Dialer.DialContext.
func (d Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialContext(ctx, network, address)
}

var _ dialer = Dialer{}
