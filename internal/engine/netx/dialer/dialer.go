package dialer

import (
	"context"
	"net"
)

// Dialer is the interface we expect from a dialer
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
