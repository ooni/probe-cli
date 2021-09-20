package measurex

import (
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Logger is the logger type we use. This type is compatible
// with the logger type of github.com/apex/log.
type Logger interface {
	netxlite.Logger

	Info(msg string)
	Infof(format string, v ...interface{})
}
