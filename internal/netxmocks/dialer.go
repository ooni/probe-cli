package netxmocks

import (
	"context"
	"net"
)

// Dialer is a mockable Dialer.
type Dialer struct {
	MockDialContext func(ctx context.Context, network, address string) (net.Conn, error)
}

// DialContext calls MockDialContext.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialContext(ctx, network, address)
}
