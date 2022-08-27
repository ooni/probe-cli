package main

//
// TCP connect (and optionally TLS handshake) measurements
//

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ctrlTCPResult is the result of the TCP check performed by the test helper.
type ctrlTCPResult = webconnectivity.ControlTCPConnectResult

// ctrlTLSResult is the result of the TLS check performed by the test helper.
type ctrlTLSResult = webconnectivity.ControlTLSHandshakeResult

// tcpResultPair contains the endpoint and the corresponding result.
type tcpResultPair struct {
	// Endpoint is the endpoint we measured.
	Endpoint string

	// TCP contains the TCP results.
	TCP ctrlTCPResult

	// TLS contains the TLS results
	TLS *ctrlTLSResult
}

// tcpConfig configures the TCP connect check.
type tcpConfig struct {
	// EnableTLS OPTIONALLY enables TLS.
	EnableTLS bool

	// Endpoint is the MANDATORY endpoint to connect to.
	Endpoint string

	// NewDialer is the MANDATORY factory for creating a new dialer.
	NewDialer func() model.Dialer

	// NewTSLHandshaker is the MANDATORY factory for creating a new handshaker.
	NewTSLHandshaker func() model.TLSHandshaker

	// Out is the MANDATORY where we'll post the TCP measurement results.
	Out chan *tcpResultPair

	// URLHostname is the MANDATORY URL.Hostname() to use.
	URLHostname string

	// Wg is MANDATORY and is used to sync with the parent.
	Wg *sync.WaitGroup
}

// tcpDo performs the TCP check.
func tcpDo(ctx context.Context, config *tcpConfig) {
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer config.Wg.Done()
	out := &tcpResultPair{
		Endpoint: config.Endpoint,
		TCP:      webconnectivity.ControlTCPConnectResult{},
		TLS:      nil, // means: not measured
	}
	defer func() {
		config.Out <- out
	}()
	dialer := config.NewDialer()
	defer dialer.CloseIdleConnections()
	conn, err := dialer.DialContext(ctx, "tcp", config.Endpoint)
	out.TCP.Failure = tcpMapFailure(newfailure(err))
	out.TCP.Status = err == nil
	defer measurexlite.MaybeClose(conn)
	if err != nil || !config.EnableTLS {
		return
	}
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
		ServerName: config.URLHostname,
	}
	thx := config.NewTSLHandshaker()
	tlsConn, _, err := thx.Handshake(ctx, conn, tlsConfig)
	out.TLS = &ctrlTLSResult{
		ServerName: config.URLHostname,
		Status:     err == nil,
		Failure:    newfailure(err),
	}
	measurexlite.MaybeClose(tlsConn)
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
