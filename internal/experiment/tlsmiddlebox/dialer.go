package tlsmiddlebox

//
// Custom TTL dialer
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const timeout time.Duration = 15 * time.Second

func NewDialerTTLWrapper() model.Dialer {
	return &dialerTTLWrapper{
		Dialer: &net.Dialer{Timeout: timeout},
	}
}

// dialerTTLWrapper wraps errors and also returns a TTL wrapped conn
type dialerTTLWrapper struct {
	Dialer model.SimpleDialer
}

var _ model.Dialer = &dialerTTLWrapper{}

// DialContext implements model.Dialer.DialContext
func (d *dialerTTLWrapper) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}
	return &dialerTTLWrapperConn{
		Conn: conn,
	}, nil
}

// CloseIdleConnections implements model.Dialer.CloseIdleConnections
func (d *dialerTTLWrapper) CloseIdleConnections() {
	// nothing to do here
}
