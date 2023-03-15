package main

//
// Logging code
//

import "github.com/ooni/probe-cli/v3/internal/model"

// prefixLogger is a logger with a prefix.
type prefixLogger struct {
	indexstr string
	logger   model.Logger
}

var _ model.Logger = &prefixLogger{}

// Debug implements DebugLogger.Debug
func (p *prefixLogger) Debug(msg string) {
	p.logger.Debug(p.indexstr + msg)
}

// Debugf implements DebugLogger.Debugf
func (p *prefixLogger) Debugf(format string, v ...interface{}) {
	p.logger.Debugf(p.indexstr+format, v...)
}

// Info implements InfoLogger.Info
func (p *prefixLogger) Info(msg string) {
	p.logger.Info(p.indexstr + msg)
}

// Infov implements InfoLogger.Infov
func (p *prefixLogger) Infof(format string, v ...interface{}) {
	p.logger.Infof(p.indexstr+format, v...)
}

// Warn implements Logger.Warn
func (p *prefixLogger) Warn(msg string) {
	p.logger.Warn(p.indexstr + msg)
}

// Warnf implements Logger.Warnf
func (p *prefixLogger) Warnf(format string, v ...interface{}) {
	p.logger.Warnf(p.indexstr+format, v...)
}
