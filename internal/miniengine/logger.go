package miniengine

//
// Emitting log messages as events.
//

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// loggerEmitter is a [model.Logger] and emits events using the given channel.
type loggerEmitter struct {
	// emitter is the channel where to emit events.
	emitter chan<- *Event

	// isVerbose indicates whether to emit debug logs.
	isVerbose bool
}

// ensure that taskLogger implements model.Logger.
var _ model.Logger = &loggerEmitter{}

// newLoggerEmitter creates a new [loggerEmitter] instance.
func newLoggerEmitter(emitter chan<- *Event, isVerbose bool) *loggerEmitter {
	return &loggerEmitter{
		emitter:   emitter,
		isVerbose: isVerbose,
	}
}

// Debug implements model.Logger.Debug.
func (cl *loggerEmitter) Debug(msg string) {
	if cl.isVerbose {
		cl.emit(EventTypeDebug, msg)
	}
}

// Debugf implements model.Logger.Debugf.
func (cl *loggerEmitter) Debugf(format string, v ...interface{}) {
	if cl.isVerbose {
		cl.Debug(fmt.Sprintf(format, v...))
	}
}

// Info implements model.Logger.Info.
func (cl *loggerEmitter) Info(msg string) {
	cl.emit(EventTypeInfo, msg)
}

// Infof implements model.Logger.Infof.
func (cl *loggerEmitter) Infof(format string, v ...interface{}) {
	cl.Info(fmt.Sprintf(format, v...))
}

// Warn implements model.Logger.Warn.
func (cl *loggerEmitter) Warn(msg string) {
	cl.emit(EventTypeWarning, msg)
}

// Warnf implements model.Logger.Warnf.
func (cl *loggerEmitter) Warnf(format string, v ...interface{}) {
	cl.Warn(fmt.Sprintf(format, v...))
}

// emit is the code that actually emits the log event.
func (cl *loggerEmitter) emit(level string, message string) {
	event := &Event{
		EventType: level,
		Message:   message,
		Progress:  0,
	}
	// Implementation note: it's fine to lose interim events
	select {
	case cl.emitter <- event:
	default:
	}
}
