package main

//
// TCP connect measurements
//

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ctrlTCPResult is the result of the TCP check performed by the test helper.
type ctrlTCPResult = webconnectivity.ControlTCPConnectResult

// tcpResultPair contains the endpoint and the corresponding result.
type tcpResultPair struct {
	// Endpoint is the endpoint we measured.
	Endpoint string

	// Result contains the results.
	Result ctrlTCPResult
}

// tcpConfig configures the TCP connect check.
type tcpConfig struct {
	// Endpoint is the MANDATORY endpoint to connect to.
	Endpoint string

	// NewDialer is the MANDATORY factory for creating a new dialer.
	NewDialer func() model.Dialer

	// Out is the MANDATORY where we'll post the TCP measurement results.
	Out chan tcpResultPair

	// Wg is MANDATORY and is used to sync with the parent.
	Wg *sync.WaitGroup
}

// tcpDo performs the TCP check.
func tcpDo(ctx context.Context, config *tcpConfig) {
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer config.Wg.Done()
	dialer := config.NewDialer()
	defer dialer.CloseIdleConnections()
	conn, err := dialer.DialContext(ctx, "tcp", config.Endpoint)
	if conn != nil {
		conn.Close()
	}
	config.Out <- tcpResultPair{
		Endpoint: config.Endpoint,
		Result: ctrlTCPResult{
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
