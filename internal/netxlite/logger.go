package netxlite

// Logger is the interface we expect from a logger.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})
}
