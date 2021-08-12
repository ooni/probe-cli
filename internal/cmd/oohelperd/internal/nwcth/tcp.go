package nwcth

import (
	"context"
	"net"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, endpoint string) (net.Conn, *TCPConnectMeasurement) {
	dialer := netx.NewDialer(netx.Config{Logger: log.Log})
	conn, err := dialer.DialContext(ctx, "tcp", endpoint)
	return conn, &TCPConnectMeasurement{
		Failure: newfailure(err),
	}
}
