package testingx

import (
	"fmt"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/logmodel"
)

// Logger implements [logmodel.Logger] and collects all the log lines.
//
// The zero value of this struct is ready to use.
type Logger struct {
	// debug contains debug lines.
	debug []string

	// info contains info lines.
	info []string

	// mu provides mutual exclusion.
	mu sync.Mutex

	// warning contains warning lines.
	warning []string
}

var _ logmodel.Logger = &Logger{}

// Debug implements logmodel.Logger.
func (l *Logger) Debug(msg string) {
	l.mu.Lock()
	l.debug = append(l.debug, msg)
	l.mu.Unlock()
}

// Debugf implements logmodel.Logger.
func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Debug(fmt.Sprintf(format, v...))
}

// Info implements logmodel.Logger.
func (l *Logger) Info(msg string) {
	l.mu.Lock()
	l.info = append(l.info, msg)
	l.mu.Unlock()
}

// Infof implements logmodel.Logger.
func (l *Logger) Infof(format string, v ...interface{}) {
	l.Info(fmt.Sprintf(format, v...))
}

// Warn implements logmodel.Logger.
func (l *Logger) Warn(msg string) {
	l.mu.Lock()
	l.warning = append(l.warning, msg)
	l.mu.Unlock()
}

// Warnf implements logmodel.Logger.
func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Warn(fmt.Sprintf(format, v...))
}

// DebugLines returns a copy of the observed debug lines.
func (l *Logger) DebugLines() []string {
	l.mu.Lock()
	out := append([]string{}, l.debug...)
	l.mu.Unlock()
	return out
}

// InfoLines returns a copy of the observed info lines.
func (l *Logger) InfoLines() []string {
	l.mu.Lock()
	out := append([]string{}, l.info...)
	l.mu.Unlock()
	return out
}

// WarnLines returns a copy of the observed warn lines.
func (l *Logger) WarnLines() []string {
	l.mu.Lock()
	out := append([]string{}, l.warning...)
	l.mu.Unlock()
	return out
}

// ClearAll removes all the log lines collected so far.
func (l *Logger) ClearAll() {
	l.mu.Lock()
	l.debug = []string{}
	l.info = []string{}
	l.warning = []string{}
	l.mu.Unlock()
}

// AllLines returns all the collected lines.
func (l *Logger) AllLines() []string {
	return append(append(append([]string{}, l.DebugLines()...), l.InfoLines()...), l.WarnLines()...)
}
