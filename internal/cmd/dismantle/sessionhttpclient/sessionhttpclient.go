// Package sessionhttpclient creates an HTTP client for
// a measurement session. We will use this client for
// communicating with the OONI backend.
package sessionhttpclient

import (
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Config contains config for creating a new session HTTP client.
type Config struct {
	ByteCounter *bytecounter.Counter
	Logger      model.Logger
	Resolver    model.Resolver

	// optional fields
	ProxyURL *url.URL
}

// New creates a new HTTP client to be used during a measurement
// session to communicate with the OONI backend.
func New(config *Config) model.HTTPClient {
	dialer := netxlite.NewDialerWithResolver(config.Logger, config.Resolver)
	dialer = netxlite.MaybeWrapWithProxyDialer(dialer, config.ProxyURL)
	handshaker := netxlite.NewTLSHandshakerStdlib(config.Logger)
	tlsDialer := netxlite.NewTLSDialer(dialer, handshaker)
	txp := netxlite.NewHTTPTransport(config.Logger, dialer, tlsDialer)
	txp = bytecounter.MaybeWrapHTTPTransport(txp, config.ByteCounter)
	return netxlite.NewHTTPClient(txp)
}
