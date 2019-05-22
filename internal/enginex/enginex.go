// Package enginex contains ooni/probe-engine extensions.
package enginex

import (
	"github.com/fatih/color"
	"github.com/ooni/probe-engine/log"
)

// LoggerAdapter exposes an interface compatible with the logger expected
// by ooni/probe-engine, and forwards real log messages to a secondary logger
// having the same interface (compatible with apex/log).
type LoggerAdapter struct {
	// Logger is the underlying logger
	log.Logger
}

// Debug emits a debug message.
func (la LoggerAdapter) Debug(msg string) {
	la.Logger.Debug(color.WhiteString("engine") + ": " + msg)
}

// Debugf formats and emits a debug message.
func (la LoggerAdapter) Debugf(format string, v ...interface{}) {
	la.Logger.Debugf(color.WhiteString("engine")+": "+format, v...)
}

// Info emits an informational message.
func (la LoggerAdapter) Info(msg string) {
	la.Logger.Info(color.BlueString("engine") + ": " + msg)
}

// Infof format and emits an informational message.
func (la LoggerAdapter) Infof(format string, v ...interface{}) {
	la.Logger.Infof(color.BlueString("engine")+": "+format, v...)
}

// Warn emits a warning message.
func (la LoggerAdapter) Warn(msg string) {
	la.Logger.Warn(color.RedString("engine") + ": " + msg)
}

// Warnf formats and emits a warning message.
func (la LoggerAdapter) Warnf(format string, v ...interface{}) {
	la.Logger.Warnf(color.RedString("engine")+": "+format, v...)
}
