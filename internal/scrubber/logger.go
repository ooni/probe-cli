package scrubber

import "fmt"

// UnderlyingLogger defines the common interface that a logger should have. It is
// out of the box compatible with `log.Log` in `apex/log`.
type UnderlyingLogger interface {
	// Debug emits a debug message.
	Debug(msg string)

	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Info emits an informational message.
	Info(msg string)

	// Infof formats and emits an informational message.
	Infof(format string, v ...interface{})

	// Warn emits a warning message.
	Warn(msg string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}

// Logger is a Logger with scrubbing. All messages are scrubbed
// including the ones that won't be emitted. As such, this logger
// is less efficient than a logger without scrubbing.
type Logger struct {
	UnderlyingLogger
}

// Debug scrubs and emits a debug message.
func (sl *Logger) Debug(message string) {
	sl.UnderlyingLogger.Debug(Scrub(message))
}

// Debugf scrubs, formats, and emits a debug message.
func (sl *Logger) Debugf(format string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(format, v...))
}

// Info scrubs and emits an informational message.
func (sl *Logger) Info(message string) {
	sl.UnderlyingLogger.Info(Scrub(message))
}

// Infof scrubs, formats, and emits an informational message.
func (sl *Logger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

// Warn scrubs and emits a warning message.
func (sl *Logger) Warn(message string) {
	sl.UnderlyingLogger.Warn(Scrub(message))
}

// Warnf scrubs, formats, and emits a warning message.
func (sl *Logger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}
