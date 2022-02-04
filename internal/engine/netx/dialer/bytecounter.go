package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// byteCounterDialer is a byte-counting-aware dialer. To perform byte counting, you
// should make sure that you insert this dialer in the dialing chain.
type byteCounterDialer struct {
	model.Dialer
}

// DialContext implements Dialer.DialContext
func (d *byteCounterDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	conn = bytecounter.WrapWithContextByteCounters(ctx, conn)
	return conn, nil
}
