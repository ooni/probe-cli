package internal

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Dialer creates net.Conn instances where (1) we delay writes if
// a delay is configured and (2) we split outgoing buffers if there
// is a configured splitter function.
type Dialer struct {
	model.Dialer
	Delay    time.Duration
	Splitter func([]byte) [][]byte
}

// DialContext implements netx.Dialer.DialContext.
func (d Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	conn = SleeperWriter{Conn: conn, Delay: d.Delay}
	conn = SplitterWriter{Conn: conn, Splitter: d.Splitter}
	return conn, nil
}
