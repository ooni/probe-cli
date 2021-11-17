package ptx

import (
	"context"
	"fmt"
	"net"
)

// FakeDialer is a fake pluggable transport dialer. It actually
// just creates a TCP connection with the given address.
type FakeDialer struct {
	// Address is the real destination address.
	Address string
}

var _ PTDialer = &FakeDialer{}

// DialContext establishes a TCP connection with d.Address.
func (d *FakeDialer) DialContext(ctx context.Context) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp", d.Address)
}

// AsBridgeArgument returns the argument to be passed to
// the tor command line to declare this bridge.
func (d *FakeDialer) AsBridgeArgument() string {
	return fmt.Sprintf("fake %s", d.Address)
}

// Name returns the pluggable transport name.
func (d *FakeDialer) Name() string {
	return "fake"
}
