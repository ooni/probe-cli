package main

//
// Logger
//

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
)

// taskLogger implements model.Logger for tasks.
type taskLogger struct {
	// emitter is used to emit log events.
	emitter taskMaybeEmitter

	// verbose indicates whether verbose logging is enabled.
	verbose bool
}

// newLogger creates a new taskLogger instance using
// the [emitter] to emit log events.
func newTaskLogger(emitter taskMaybeEmitter) *taskLogger {
	return &taskLogger{
		emitter: emitter,
		verbose: false,
	}
}

var _ model.Logger = &taskLogger{}

// Debugf implements model.Logger.Debugf.
func (tl *taskLogger) Debugf(format string, values ...any) {
	if tl.verbose {
		tl.emit(abi.LogLevel_DEBUG, fmt.Sprintf(format, values...))
	}
}

// Debug implements model.Logger.Debug.
func (tl *taskLogger) Debug(message string) {
	if tl.verbose {
		tl.emit(abi.LogLevel_DEBUG, message)
	}
}

// Infof implements model.Logger.Infof.
func (tl *taskLogger) Infof(format string, values ...any) {
	tl.emit(abi.LogLevel_INFO, fmt.Sprintf(format, values...))
}

// Info implements model.Logger.Info.
func (tl *taskLogger) Info(message string) {
	tl.emit(abi.LogLevel_INFO, message)
}

// Warnf implements model.Logger.Warnf.
func (tl *taskLogger) Warnf(format string, values ...any) {
	tl.emit(abi.LogLevel_WARNING, fmt.Sprintf(format, values...))
}

// Warn implements model.Logger.Warn.
func (tl *taskLogger) Warn(message string) {
	tl.emit(abi.LogLevel_WARNING, message)
}

// emit emits a log message.
func (tl *taskLogger) emit(level abi.LogLevel, message string) {
	value := &abi.LogEvent{
		Level:   level,
		Message: message,
	}
	tl.emitter.maybeEmitEvent("Log", value)
}
