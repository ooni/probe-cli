package webconnectivity

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	Dialer   model.Dialer
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
			Failure: tcpMapFailure(newfailure(err)),
			Status:  err == nil,
		},
	}
}

// tcpMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L392
func tcpMapFailure(failure *string) *string {
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureGenericTimeoutError:
			return failure // already using the same name
		case netxlite.FailureConnectionRefused:
			s := "connection_refused_error"
			return &s
		default:
			// The definition of this error according to Twisted is
			// "something went wrong when connecting". Because we are
			// indeed basically just connecting here, it seems safe
			// to map any other error to "connect_error" here.
			s := "connect_error"
			return &s
		}
	}
}
