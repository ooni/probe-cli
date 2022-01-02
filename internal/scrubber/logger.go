package scrubber

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Logger is a Logger with scrubbing. All messages are scrubbed
// including the ones that won't be emitted. As such, this logger
// is less efficient than a logger without scrubbing.
type Logger struct {
	model.Logger
}

// Debug scrubs and emits a debug message.
func (sl *Logger) Debug(message string) {
	sl.Logger.Debug(Scrub(message))
}

// Debugf scrubs, formats, and emits a debug message.
func (sl *Logger) Debugf(format string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(format, v...))
}

// Info scrubs and emits an informational message.
func (sl *Logger) Info(message string) {
	sl.Logger.Info(Scrub(message))
}

// Infof scrubs, formats, and emits an informational message.
func (sl *Logger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

// Warn scrubs and emits a warning message.
func (sl *Logger) Warn(message string) {
	sl.Logger.Warn(Scrub(message))
}

// Warnf scrubs, formats, and emits a warning message.
func (sl *Logger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}
