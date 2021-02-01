// +build !shaping

package dialer

import (
	"context"
	"net"
)

// ShapingDialer ensures we don't use too much bandwidth
// when using integration tests at GitHub. To select
// the implementation with shaping use `-tags shaping`.
type ShapingDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d ShapingDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return d.Dialer.DialContext(ctx, network, address)
}
