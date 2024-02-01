// Package logmodel contains the core log model.
//
// This package has been separated from the ./internal/model package
// so that ./internal/cmd/buildtool can support go1.18+.
//
// See https://github.com/ooni/probe/issues/2664 for context.
package logmodel

// DebugLogger is a logger emitting only debug messages.
type DebugLogger interface {
	// Debug emits a debug message.
	Debug(msg string)

	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})
}

// InfoLogger is a logger emitting debug and infor messages.
type InfoLogger interface {
	// An InfoLogger is also a DebugLogger.
	DebugLogger

	// Info emits an informational message.
	Info(msg string)

	// Infof formats and emits an informational message.
	Infof(format string, v ...interface{})
}

// Logger defines the common interface that a logger should have. It is
// out of the box compatible with `log.Log` in `apex/log`.
type Logger interface {
	// A Logger is also an InfoLogger.
	InfoLogger

	// Warn emits a warning message.
	Warn(msg string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}
