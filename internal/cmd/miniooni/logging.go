package main

//
// Logging functionality
//

import (
	"fmt"
	"io"
	"time"

	"github.com/apex/log"
)

// logStartTime is the time when we started logging
var logStartTime = time.Now()

// logHandler implements the log handler required by github.com/apex/log
type logHandler struct {
	// Writer is the underlying writer
	io.Writer
}

var _ log.Handler = &logHandler{}

// HandleLog implements log.Handler
func (h *logHandler) HandleLog(e *log.Entry) (err error) {
	s := fmt.Sprintf("[%14.6f] <%s> %s", time.Since(logStartTime).Seconds(), e.Level, e.Message)
	if len(e.Fields) > 0 {
		s += fmt.Sprintf(": %+v", e.Fields)
	}
	s += "\n"
	_, err = h.Writer.Write([]byte(s))
	return
}
