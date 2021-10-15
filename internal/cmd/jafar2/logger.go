package main

import (
	"fmt"
	"os"
	"time"

	"github.com/apex/log"
)

// LoggerHandler is the apex/log Handler we use.
type LoggerHandler struct {
	begin time.Time
}

// NewLoggerHandler creates a new LoggerHandler instance.
func NewLoggerHandler() *LoggerHandler {
	return &LoggerHandler{begin: time.Now()}
}

// HandleLog emits a log message.
func (lh *LoggerHandler) HandleLog(e *log.Entry) error {
	t := time.Since(lh.begin).Seconds()
	message := fmt.Sprintf("[%10.6f] <%s> %s\n", t, e.Level, e.Message)
	_, err := os.Stderr.WriteString(message)
	return err
}
