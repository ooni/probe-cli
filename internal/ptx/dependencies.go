package ptx

import (
	"context"
	"net"
)

// UnderlyingDialer is the underlying dialer used for dialing.
type UnderlyingDialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Logger allows us to log messages.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Infof formats and emits an informational message.
	Infof(format string, v ...interface{})

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// silentLogger implements Logger.
type silentLogger struct{}

// Debugf implements Logger.Debugf.
func (*silentLogger) Debugf(format string, v ...interface{}) {}

// Infof implements Logger.Infof.
func (*silentLogger) Infof(format string, v ...interface{}) {}

// Warnf implements Logger.Warnf.
func (*silentLogger) Warnf(format string, v ...interface{}) {}

// defaultLogger is the default silentLogger instance.
var defaultLogger Logger = &silentLogger{}
