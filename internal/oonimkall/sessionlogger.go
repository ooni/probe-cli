package oonimkall

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type loggerVerbose struct {
	Logger
}

func (slv loggerVerbose) Debugf(format string, v ...interface{}) {
	slv.Debug(fmt.Sprintf(format, v...))
}
func (slv loggerVerbose) Infof(format string, v ...interface{}) {
	slv.Info(fmt.Sprintf(format, v...))
}
func (slv loggerVerbose) Warnf(format string, v ...interface{}) {
	slv.Warn(fmt.Sprintf(format, v...))
}

type loggerNormal struct {
	Logger
}

func (sln loggerNormal) Debugf(format string, v ...interface{}) {
	// nothing
}
func (sln loggerNormal) Debug(msg string) {
	// nothing
}
func (sln loggerNormal) Infof(format string, v ...interface{}) {
	sln.Info(fmt.Sprintf(format, v...))
}
func (sln loggerNormal) Warnf(format string, v ...interface{}) {
	sln.Warn(fmt.Sprintf(format, v...))
}

type loggerQuiet struct{}

func (loggerQuiet) Debugf(format string, v ...interface{}) {
	// nothing
}
func (loggerQuiet) Debug(msg string) {
	// nothing
}
func (loggerQuiet) Infof(format string, v ...interface{}) {
	// nothing
}
func (loggerQuiet) Info(msg string) {
	// nothing
}
func (loggerQuiet) Warnf(format string, v ...interface{}) {
	// nothing
}
func (loggerQuiet) Warn(msg string) {
	// nothing
}

func newLogger(logger Logger, verbose bool) model.Logger {
	if logger == nil {
		return loggerQuiet{}
	}
	if verbose {
		return loggerVerbose{Logger: logger}
	}
	return loggerNormal{Logger: logger}
}
