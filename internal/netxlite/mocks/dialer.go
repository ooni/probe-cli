package mocks

import (
	"context"
	"net"
)

// Dialer is a mockable Dialer.
type Dialer struct {
	MockDialContext          func(ctx context.Context, network, address string) (net.Conn, error)
	MockCloseIdleConnections func()
}

// DialContext calls MockDialContext.
func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.MockDialContext(ctx, network, address)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (d *Dialer) CloseIdleConnections() {
	d.MockCloseIdleConnections()
}
