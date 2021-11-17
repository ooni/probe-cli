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
	// Rationale for using NoLogger here: previously this code did
	// not use a logger as well, so it's fine to keep it as is.
	return netxlite.NewHTTP3Transport(&NoLogger{},
		netxlite.NewQUICDialerFromContextDialerAdapter(config.QUICDialer),
		config.TLSConfig)
}

type NoLogger struct{}

func (*NoLogger) Debug(message string) {}

func (*NoLogger) Debugf(format string, v ...interface{}) {}
