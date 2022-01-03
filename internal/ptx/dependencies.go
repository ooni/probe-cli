package ptx

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// UnderlyingDialer is the underlying dialer used for dialing.
type UnderlyingDialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// defaultLogger is the default silentLogger instance.
var defaultLogger model.Logger = model.DiscardLogger
