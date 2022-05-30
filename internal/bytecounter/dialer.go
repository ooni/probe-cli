package bytecounter

//
// model.Dialer wrappers
//

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ContextAwareDialer is a model.Dialer that attempts to count bytes using
// the MaybeWrapWithContextByteCounters function.
//
// Bug
//
// This implementation cannot properly account for the bytes that are sent by
// persistent connections, because they stick to the counters set when the
// connection was established. This typically means we miss the bytes sent and
// received when submitting a measurement. Such bytes are specifically not
// seen by the experiment specific byte counter.
//
// For this reason, this implementation may be heavily changed/removed.
type ContextAwareDialer struct {
	Dialer model.Dialer
}

// NewContextAwareDialer creates a new ContextAwareDialer.
func NewContextAwareDialer(dialer model.Dialer) *ContextAwareDialer {
	return &ContextAwareDialer{Dialer: dialer}
}

var _ model.Dialer = &ContextAwareDialer{}

// DialContext implements Dialer.DialContext
func (d *ContextAwareDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	conn = MaybeWrapWithContextByteCounters(ctx, conn)
	return conn, nil
}

// CloseIdleConnections implements Dialer.CloseIdleConnections.
func (d *ContextAwareDialer) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}
