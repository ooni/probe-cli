package session

import (
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// LogEvent is an event emitted by a logger.
type LogEvent struct {
	// Timestamp is the log timestamp.
	Timestamp time.Time

	// Level is the log level.
	Level string

	// Message is the log message.
	Message string
}

// newLogger creates a [model.Logger] that emits
// [LogEvent] events using the [Session].
func (s *Session) newLogger(verbose bool) model.Logger {
	return &sessionLogger{
		session: s,
		verbose: verbose,
	}
}

// sessionLogger is a [model.Logger] using a [Session].
type sessionLogger struct {
	session *Session
	verbose bool
}

// Debug implements model.Logger
func (sl *sessionLogger) Debug(msg string) {
	if sl.verbose {
		sl.emit("DEBUG", msg)
	}
}

// Debugf implements model.Logger
func (sl *sessionLogger) Debugf(format string, v ...interface{}) {
	if sl.verbose {
		sl.Debug(fmt.Sprintf(format, v...))
	}
}

// Info implements model.Logger
func (sl *sessionLogger) Info(msg string) {
	sl.emit("INFO", msg)
}

// Infof implements model.Logger
func (sl *sessionLogger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

// Warn implements model.Logger
func (sl *sessionLogger) Warn(msg string) {
	sl.emit("WARNING", msg)
}

// Warnf implements model.Logger
func (sl *sessionLogger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}

// emit emits a log message.
func (sl *sessionLogger) emit(level, message string) {
	ev := &Event{
		Log: &LogEvent{
			Timestamp: time.Now(),
			Level:     level,
			Message:   message,
		},
	}
	sl.session.emit(ev)
}
