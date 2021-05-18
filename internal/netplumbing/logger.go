package netplumbing

import "context"

// Logger formats and emits log messages.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Debug emits a debug message.
	Debug(message string)
}

// quietLogger is a logger that doesn't emit any message.
type quietLogger struct{}

// Debugf implements Logger.Debugf.
func (*quietLogger) Debugf(format string, v ...interface{}) {}

// Debug implements Logger.Debug.
func (*quietLogger) Debug(message string) {}

// defaultLogger is the default logger.
var defaultLogger = &quietLogger{}

// logger returns the configured logger or the DefaultLogger.
func (txp *Transport) logger(ctx context.Context) Logger {
	if config := ContextConfig(ctx); config != nil && config.Logger != nil {
		return config.Logger
	}
	return defaultLogger
}
