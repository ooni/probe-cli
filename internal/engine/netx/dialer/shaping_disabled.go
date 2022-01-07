//go:build !shaping
// +build !shaping

package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// shapingDialer ensures we don't use too much bandwidth
// when using integration tests at GitHub. To select
// the implementation with shaping use `-tags shaping`.
type shapingDialer struct {
	model.Dialer
}

// DialContext implements Dialer.DialContext
func (d *shapingDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return d.Dialer.DialContext(ctx, network, address)
}
