package main

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type LogLevel string

const (
	// The DEBUG log level.
	logLevel_DEBUG LogLevel = "DEBUG"

	// The INFO log level.
	logLevel_INFO LogLevel = "INFO"

	// The WARNING log level.
	logLevel_WARNING LogLevel = "WARNING"
)

type logResponse struct {
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
// the [emitter] to emit log events.
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
		tl.emit(logLevel_DEBUG, fmt.Sprintf(format, values...))
	}
}

// Debug implements model.Logger.Debug.
func (tl *taskLogger) Debug(message string) {
	if tl.verbose {
		tl.emit(logLevel_DEBUG, message)
	}
}

// Infof implements model.Logger.Infof.
func (tl *taskLogger) Infof(format string, values ...any) {
	tl.emit(logLevel_INFO, fmt.Sprintf(format, values...))
}

// Info implements model.Logger.Info.
func (tl *taskLogger) Info(message string) {
	tl.emit(logLevel_INFO, message)
}

// Warnf implements model.Logger.Warnf.
func (tl *taskLogger) Warnf(format string, values ...any) {
	tl.emit(logLevel_WARNING, fmt.Sprintf(format, values...))
}

// Warn implements model.Logger.Warn.
func (tl *taskLogger) Warn(message string) {
	tl.emit(logLevel_WARNING, message)
}

// emit emits a log message.
func (tl *taskLogger) emit(level LogLevel, message string) {
	logResp := &response{
		Logger: logResponse{
			Level:   level,
			Message: message,
		},
	}
	tl.emitter.maybeEmitEvent(logResp)
}
