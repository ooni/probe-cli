package model

//
// Common interface for circumvention tunnels.
//

import (
	"context"
	"errors"
)

// TunnelBootstrapEvent is an event emitted during the bootstrap.
type TunnelBootstrapEvent struct {
	// Progress is the progress we have made so far as a number between 0 and 1.
	Progress float64

	// Message is the corresponding explanatory message.
	Message string
}

// Tunnel is a tunnel for communicating with the OONI backend.
type Tunnel interface {
	// LookupProbeIP discovers the probe's IP address using this tunnel.
	LookupProbeIP(ctx context.Context) (string, error)

	// Name returns the tunnel name.
	Name() string

	// NewHTTPTransport returns a new HTTP transport using this tunnel.
	NewHTTPTransport(logger Logger) (HTTPTransport, error)

	// NewDNSOverHTTPSResolver returns a new DNS-over-HTTPS resolver using this tunnel.
	NewDNSOverHTTPSResolver(logger Logger, URL string) (Resolver, error)

	// Start starts the tunnel and returns two channels. The first channel gets
	// interim bootstrap events, the second gets the final result. If the context
	// is cancelled or expires during the bootstrap, we interrupt the bootstrap
	// early and return an error via the error channel. If you already started
	// the tunnel, this function posts a nil error on the second channel.
	Start(ctx context.Context) (<-chan *TunnelBootstrapEvent, <-chan error)

	// Stop stops the tunnel.
	Stop()
}

// ErrTunnelNotStarted indicates we have not started the tunnel.
var ErrTunnelNotStarted = errors.New("tunnel: not started")
