package nwcth

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/nwebconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// CtrlTCPResult is the result of the TCP check performed by the test helper.
type CtrlTCPResult = nwebconnectivity.ControlTCPConnect

// TCPConfig configures the TCP connect check.
type TCPConfig struct {
	Dialer   netx.Dialer
	Endpoint string
}

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, config *TCPConfig) (net.Conn, *CtrlTCPResult) {
	conn, err := config.Dialer.DialContext(ctx, "tcp", config.Endpoint)
	return conn, &CtrlTCPResult{
		Failure: newfailure(err),
	}
}
