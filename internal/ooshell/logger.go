package ooshell

//
// logger.go
//
// Contains code for logging.
//

import (
	"fmt"
	"io"
	"time"

	"github.com/apex/log"
)

// Logger is the logger used by this package.
type Logger interface {
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

// NewLogger creates a new apex/log.Logger instance using as
// handler the default handler used by this package.
func NewLogger(w io.Writer) *log.Logger {
	return &log.Logger{Level: log.InfoLevel, Handler: NewLogHandler(w)}
}

// NewLogHandler returns the apex/log.Handler instance we
// use by default for logging in this package.
func NewLogHandler(w io.Writer) log.Handler {
	return &logHandler{begin: time.Now(), w: w}
}

type logHandler struct {
	begin time.Time
	w     io.Writer
}

func (h *logHandler) HandleLog(e *log.Entry) (err error) {
	s := fmt.Sprintf("[%14.6f] <%s> %s", time.Since(h.begin).Seconds(), e.Level, e.Message)
	if len(e.Fields) > 0 {
		s += fmt.Sprintf(": %+v", e.Fields)
	}
	s += "\n"
	_, err = h.w.Write([]byte(s))
	return
}
