package motor

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type LogLevel string

const (
	// The DEBUG log level.
	logDebug LogLevel = "DEBUG"

	// The INFO log level.
	logInfo LogLevel = "INFO"

	// The WARNING log level.
	logWarning LogLevel = "WARNING"
)

// LogResponse is the response for any logging task.
type LogResponse struct {
	Level   LogLevel `json:",omitempty"`
	Message string   `json:",omitempty"`
}

// taskLogger implements model.Logger for tasks.
type taskLogger struct {
	// emitter is used to emit log events.
	emitter taskMaybeEmitter

	// verbose indicates whether verbose logging is enabled.
	verbose bool
}

// newLogger creates a new taskLogger instance using
// the emitter to emit log events.
func newTaskLogger(emitter taskMaybeEmitter, verbose bool) *taskLogger {
	return &taskLogger{
		emitter: emitter,
		verbose: verbose,
	}
}

var _ model.Logger = &taskLogger{}

// Debugf implements model.Logger.Debugf.
func (tl *taskLogger) Debugf(format string, values ...any) {
	if tl.verbose {
		tl.emit(logDebug, fmt.Sprintf(format, values...))
	}
}

// Debug implements model.Logger.Debug.
func (tl *taskLogger) Debug(message string) {
	if tl.verbose {
		tl.emit(logDebug, message)
	}
}

// Infof implements model.Logger.Infof.
func (tl *taskLogger) Infof(format string, values ...any) {
	tl.emit(logInfo, fmt.Sprintf(format, values...))
}

// Info implements model.Logger.Info.
func (tl *taskLogger) Info(message string) {
	tl.emit(logInfo, message)
}

// Warnf implements model.Logger.Warnf.
func (tl *taskLogger) Warnf(format string, values ...any) {
	tl.emit(logWarning, fmt.Sprintf(format, values...))
}

// Warn implements model.Logger.Warn.
func (tl *taskLogger) Warn(message string) {
	tl.emit(logWarning, message)
}

// emit emits a log message.
func (tl *taskLogger) emit(level LogLevel, message string) {
	logResp := &Response{
		Logger: LogResponse{
			Level:   level,
			Message: message,
		},
	}
	tl.emitter.maybeEmitEvent(logResp)
}
