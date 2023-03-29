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
	// ByteCounter is the MANDATORY byte counter to use.
	ByteCounter *bytecounter.Counter

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// ProxyURL is the OPTIONAL proxy URL that the HTTPClient
	// returned by New should be using.
	ProxyURL *url.URL

	// Resolver is the MANDATORY resolver to use.
	Resolver model.Resolver
}

// New creates a new HTTPClient to be used during a measurement
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
