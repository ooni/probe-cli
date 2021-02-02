package tasks

import (
	"fmt"
)

// ChanLogger is a logger targeting a channel
type ChanLogger struct {
	emitter    *EventEmitter
	hasdebug   bool
	hasinfo    bool
	haswarning bool
	out        chan<- *Event
}

// Debug implements Logger.Debug
func (cl *ChanLogger) Debug(msg string) {
	if cl.hasdebug {
		cl.emitter.Emit("log", EventLog{
			LogLevel: "DEBUG",
			Message:  msg,
		})
	}
}

// Debugf implements Logger.Debugf
func (cl *ChanLogger) Debugf(format string, v ...interface{}) {
	if cl.hasdebug {
		cl.Debug(fmt.Sprintf(format, v...))
	}
}

// Info implements Logger.Info
func (cl *ChanLogger) Info(msg string) {
	if cl.hasinfo {
		cl.emitter.Emit("log", EventLog{
			LogLevel: "INFO",
			Message:  msg,
		})
	}
}

// Infof implements Logger.Infof
func (cl *ChanLogger) Infof(format string, v ...interface{}) {
	if cl.hasinfo {
		cl.Info(fmt.Sprintf(format, v...))
	}
}

// Warn implements Logger.Warn
func (cl *ChanLogger) Warn(msg string) {
	if cl.haswarning {
		cl.emitter.Emit("log", EventLog{
			LogLevel: "WARNING",
			Message:  msg,
		})
	}
}

// Warnf implements Logger.Warnf
func (cl *ChanLogger) Warnf(format string, v ...interface{}) {
	if cl.haswarning {
		cl.Warn(fmt.Sprintf(format, v...))
	}
}

// NewChanLogger creates a new ChanLogger instance.
func NewChanLogger(emitter *EventEmitter, logLevel string,
	out chan<- *Event) *ChanLogger {
	cl := &ChanLogger{
		emitter: emitter,
		out:     out,
	}
	switch logLevel {
	case "DEBUG", "DEBUG2":
		cl.hasdebug = true
		fallthrough
	case "INFO":
		cl.hasinfo = true
		fallthrough
	case "ERR", "WARNING":
		fallthrough
	default:
		cl.haswarning = true
	}
	return cl
}
