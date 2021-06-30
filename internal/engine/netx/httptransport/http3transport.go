package httptransport

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewHTTP3Transport creates a new HTTP3Transport instance.
//
// Deprecation warning
//
// New code should use netxlite.NewHTTP3Transport instead.
func NewHTTP3Transport(config Config) RoundTripper {
	return netxlite.NewHTTP3Transport(config.QUICDialer, config.TLSConfig)
}
