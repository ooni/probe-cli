package torcontrolalgo

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// Conn is the control conn abstraction used by this package.
type Conn interface {
	// Notifications returns the channel from which one could read
	// the asynchronous events emitted by tor.
	Notifications() <-chan *torcontrolnet.Response

	// SendRecv sends a sync request and returns the corresponding
	// response returned by tor or an error.
	SendRecv(ctx context.Context, format string, args ...any) (*torcontrolnet.Response, error)
}
