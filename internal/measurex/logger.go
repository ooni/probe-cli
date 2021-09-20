package measurex

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Logger is the logger type we use.
type Logger interface {
	netxlite.Logger

	Info(msg string)
	Infof(format string, v ...interface{})
}
