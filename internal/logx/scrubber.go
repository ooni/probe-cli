package logx

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

// ScrubberLogger is a [model.Logger] with scrubbing. All messages are scrubbed including the ones
// that won't be emitted. As such, this logger is less efficient than a logger without scrubbing.
//
// The zero value is invalid; please init all MANDATORY fields.
type ScrubberLogger struct {
	// Logger is the MANDATORY underlying logger to use.
	Logger model.Logger
}

// Debug scrubs and emits a debug message.
func (sl *ScrubberLogger) Debug(message string) {
	sl.Logger.Debug(scrubber.ScrubString(message))
}

// Debugf scrubs, formats, and emits a debug message.
func (sl *ScrubberLogger) Debugf(format string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(format, v...))
}

// Info scrubs and emits an informational message.
func (sl *ScrubberLogger) Info(message string) {
	sl.Logger.Info(scrubber.ScrubString(message))
}

// Infof scrubs, formats, and emits an informational message.
func (sl *ScrubberLogger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

// Warn scrubs and emits a warning message.
func (sl *ScrubberLogger) Warn(message string) {
	sl.Logger.Warn(scrubber.ScrubString(message))
}

// Warnf scrubs, formats, and emits a warning message.
func (sl *ScrubberLogger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}
