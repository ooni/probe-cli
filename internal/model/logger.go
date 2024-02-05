package model

import "github.com/ooni/probe-cli/v3/internal/logmodel"

//
// Logger
//

// DebugLogger is a logger emitting only debug messages.
type DebugLogger = logmodel.DebugLogger

// InfoLogger is a logger emitting debug and infor messages.
type InfoLogger = logmodel.InfoLogger

// Logger defines the common interface that a logger should have. It is
// out of the box compatible with `log.Log` in `apex/log`.
type Logger = logmodel.Logger

// DiscardLogger is the default logger that discards its input
var DiscardLogger Logger = logDiscarder{}

// logDiscarder is a logger that discards its input
type logDiscarder struct{}

// Debug implements DebugLogger.Debug
func (logDiscarder) Debug(msg string) {}

// Debugf implements DebugLogger.Debugf
func (logDiscarder) Debugf(format string, v ...interface{}) {}

// Info implements InfoLogger.Info
func (logDiscarder) Info(msg string) {}

// Infov implements InfoLogger.Infov
func (logDiscarder) Infof(format string, v ...interface{}) {}

// Warn implements Logger.Warn
func (logDiscarder) Warn(msg string) {}

// Warnf implements Logger.Warnf
func (logDiscarder) Warnf(format string, v ...interface{}) {}

// ErrorToStringOrOK emits "ok" on "<nil>"" values for success.
func ErrorToStringOrOK(err error) string {
	if err != nil {
		return err.Error()
	}
	return "ok"
}

// ValidLoggerOrDefault is a factory that either returns the logger
// provided as argument, if not nil, or DiscardLogger.
func ValidLoggerOrDefault(logger Logger) Logger {
	if logger != nil {
		return logger
	}
	return DiscardLogger
}
