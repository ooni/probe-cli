package webconnectivity

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

// CtrlTCPResult is the result of the TCP check performed by the test helper.
type CtrlTCPResult = webconnectivity.ControlTCPConnectResult

// TCPResultPair contains the endpoint and the corresponding result.
type TCPResultPair struct {
	Endpoint string
	Result   CtrlTCPResult
}

// TCPConfig configures the TCP connect check.
type TCPConfig struct {
	Dialer   netx.Dialer
	Endpoint string
	Out      chan TCPResultPair
	Wg       *sync.WaitGroup
}

// TCPDo performs the TCP check.
func TCPDo(ctx context.Context, config *TCPConfig) {
	defer config.Wg.Done()
	conn, err := config.Dialer.DialContext(ctx, "tcp", config.Endpoint)
	if conn != nil {
		conn.Close()
	}
	config.Out <- TCPResultPair{
		Endpoint: config.Endpoint,
		Result: CtrlTCPResult{
			Failure: newfailure(err),
			Status:  err == nil,
		},
	}
}
