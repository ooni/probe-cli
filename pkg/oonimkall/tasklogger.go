package oonimkall

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
)

//
// This file implements the logger used by a task. Outside
// of this file, the rest of the codebase just sees a generic
// model.Logger that can log events.
//

// taskLogger is the logger used by a task.
type taskLogger struct {
	// emitter is the event emitter.
	emitter taskEmitter

	// hasDebug indicates whether to emit debug logs.
	hasDebug bool

	// hasInfo indicates whether to emit info logs.
	hasInfo bool

	// hasWarning indicates whether to emit warning logs.
	hasWarning bool
}

// ensure that taskLogger implements model.Logger.
var _ model.Logger = &taskLogger{}

// Debug implements model.Logger.Debug.
func (cl *taskLogger) Debug(msg string) {
	if cl.hasDebug {
		cl.emit(logLevelDebug, msg)
	}
}

// Debugf implements model.Logger.Debugf.
func (cl *taskLogger) Debugf(format string, v ...interface{}) {
	if cl.hasDebug {
		cl.Debug(fmt.Sprintf(format, v...))
	}
}

// Info implements model.Logger.Info.
func (cl *taskLogger) Info(msg string) {
	if cl.hasInfo {
		cl.emit(logLevelInfo, msg)
	}
}

// Infof implements model.Logger.Infof.
func (cl *taskLogger) Infof(format string, v ...interface{}) {
	if cl.hasInfo {
		cl.Info(fmt.Sprintf(format, v...))
	}
}

// Warn implements model.Logger.Warn.
func (cl *taskLogger) Warn(msg string) {
	if cl.hasWarning {
		cl.emit(logLevelWarning, msg)
	}
}

// Warnf implements model.Logger.Warnf.
func (cl *taskLogger) Warnf(format string, v ...interface{}) {
	if cl.hasWarning {
		cl.Warn(fmt.Sprintf(format, v...))
	}
}

// emit is the code that actually emits the log event.
func (cl *taskLogger) emit(level string, message string) {
	cl.emitter.Emit(eventTypeLog, eventLog{
		LogLevel: level,
		Message:  message,
	})
}

// newTaskLogger creates a new taskLogger instance.
//
// Arguments:
//
// - emitter is the emitter that will emit log events;
//
// - logLevel is the maximum log level that will be emitted.
//
// Returns:
//
// - a properly configured instance of taskLogger.
//
// Remarks:
//
// - log levels are sorted as usual: ERR is more sever than
// WARNING, WARNING is more sever than INFO, etc.
func newTaskLogger(emitter taskEmitter, logLevel string) *taskLogger {
	cl := &taskLogger{
		emitter: emitter,
	}
	switch logLevel {
	case logLevelDebug, logLevelDebug2:
		cl.hasDebug = true
		fallthrough
	case logLevelInfo:
		cl.hasInfo = true
		fallthrough
	case logLevelErr, logLevelWarning:
		fallthrough
	default:
		cl.hasWarning = true
	}
	return cl
}
