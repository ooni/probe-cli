package model

// Logger defines the common interface that a logger should have. It is
// out of the box compatible with `log.Log` in `apex/log`.
type Logger interface {
	// Debug emits a debug message.
	Debug(msg string)

	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Info emits an informational message.
	Info(msg string)

	// Infof format and emits an informational message.
	Infof(format string, v ...interface{})

	// Warn emits a warning message.
	Warn(msg string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// DiscardLogger is a logger that discards its input
var DiscardLogger Logger = logDiscarder{}

type logDiscarder struct{}

func (logDiscarder) Debug(msg string) {}

func (logDiscarder) Debugf(format string, v ...interface{}) {}

func (logDiscarder) Info(msg string) {}

func (logDiscarder) Infof(format string, v ...interface{}) {}

func (logDiscarder) Warn(msg string) {}

func (logDiscarder) Warnf(format string, v ...interface{}) {}
