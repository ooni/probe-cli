// Package httptransport contains HTTP transport extensions.
package httptransport

import (
	"crypto/tls"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Config contains the configuration required for constructing an HTTP transport
type Config struct {
	Dialer     model.Dialer
	QUICDialer model.QUICDialer
	TLSDialer  model.TLSDialer
	TLSConfig  *tls.Config
}

// NewHTTP3Transport creates a new HTTP3Transport instance.
//
// Deprecation warning
//
// New code should use netxlite.NewHTTP3Transport instead.
func NewHTTP3Transport(config Config) model.HTTPTransport {
	// Rationale for using NoLogger here: previously this code did
	// not use a logger as well, so it's fine to keep it as is.
	return netxlite.NewHTTP3Transport(model.DiscardLogger,
		config.QUICDialer, config.TLSConfig)
}

// NewSystemTransport creates a new "system" HTTP transport. That is a transport
// using the Go standard library with custom dialer and TLS dialer.
//
// Deprecation warning
//
// New code should use netxlite.NewHTTPTransport instead.
func NewSystemTransport(config Config) model.HTTPTransport {
	return netxlite.NewOOHTTPBaseTransport(config.Dialer, config.TLSDialer)
}
