package bytecounter

//
// model.Dialer wrappers
//

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// MaybeWrapWithContextAwareDialer wraps the given dialer with a ContextAwareDialer
// if the enabled argument is true and otherwise just returns the given dialer.
//
// Bug
//
// This implementation cannot properly account for the bytes that are sent by
// persistent connections, because they stick to the counters set when the
// connection was established. This typically means we miss the bytes sent and
// received when submitting a measurement. Such bytes are specifically not
// seen by the experiment specific byte counter.
//
// For this reason, this implementation may be heavily changed/removed
// in the future (<- this message is now ~two years old, though).
func MaybeWrapWithContextAwareDialer(enabled bool, dialer model.Dialer) model.Dialer {
	if !enabled {
		return dialer
	}
	return WrapWithContextAwareDialer(dialer)
}

// contextAwareDialer is a model.Dialer that attempts to count bytes using
// the MaybeWrapWithContextByteCounters function.
type contextAwareDialer struct {
	Dialer model.Dialer
}

// WrapWithContextAwareDialer creates a new ContextAwareDialer. See the docs
// of MaybeWrapWithContextAwareDialer for a list of caveats.
func WrapWithContextAwareDialer(dialer model.Dialer) *contextAwareDialer {
	return &contextAwareDialer{Dialer: dialer}
}

var _ model.Dialer = &contextAwareDialer{}

// DialContext implements Dialer.DialContext
func (d *contextAwareDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	conn = MaybeWrapWithContextByteCounters(ctx, conn)
	return conn, nil
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *contextAwareDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}
