package main

//
// QUIC handshake measurements
//

import (
	"context"
	"crypto/tls"
	"sync"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ctrlQUICResult is the result of the QUIC check performed by the test helper.
type ctrlQUICResult = model.THTLSHandshakeResult

// quicResult contains the endpoint and the corresponding result.
type quicResult struct {
	// Address is the IP address we measured.
	Address string

	// Endpoint is the endpoint we measured.
	Endpoint string

	// QUIC contains the QUIC results
	QUIC ctrlQUICResult
}

// quicConfig configures the QUIC connect check.
type quicConfig struct {
	// Address is the MANDATORY address to measure.
	Address string

	// Endpoint is the MANDATORY endpoint to connect to.
	Endpoint string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NewQUICDialer is the MANDATORY factory for creating a new QUIC dialer.
	NewQUICDialer func(model.Logger) model.QUICDialer

	// Out is the MANDATORY chan where we'll post the QUIC measurement results.
	Out chan *quicResult

	// URLHostname is the MANDATORY URL.Hostname() to use.
	URLHostname string

	// Wg is MANDATORY and is used to sync with the parent.
	Wg *sync.WaitGroup
}

// quicDo performs the QUIC handshake check.
func quicDo(ctx context.Context, config *quicConfig) {
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	defer config.Wg.Done()
	out := &quicResult{
		Address:  config.Address,
		Endpoint: config.Endpoint,
		QUIC:     ctrlQUICResult{},
	}
	defer func() {
		config.Out <- out
	}()
	ol := measurexlite.NewOperationLogger(
		config.Logger,
		"QUICConnect %s SNI=%s",
		config.Endpoint,
		config.URLHostname,
	)
	dialer := config.NewQUICDialer(config.Logger)
	defer dialer.CloseIdleConnections()

	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using a cached cert pool.
	tlsConfig := &tls.Config{
		NextProtos: []string{"h3"},
		RootCAs:    certpool,
		ServerName: config.URLHostname,
	}
	quicConn, err := dialer.DialContext(ctx, config.Endpoint, tlsConfig, &quic.Config{})
	defer measurexlite.MaybeCloseQUICConn(quicConn)
	ol.Stop(err)

	out.QUIC = ctrlQUICResult{
		ServerName: config.URLHostname,
		Status:     err == nil,
		Failure:    newfailure(err),
	}
}
