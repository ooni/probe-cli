package main

import "github.com/ooni/probe-cli/v3/internal/model"

// indexLogger is a logger with an index.
type indexLogger struct {
	indexstr string
	logger   model.Logger
}

var _ model.Logger = &indexLogger{}

// Debug implements DebugLogger.Debug
func (p *indexLogger) Debug(msg string) {
	p.logger.Debug(p.indexstr + msg)
}

// Debugf implements DebugLogger.Debugf
func (p *indexLogger) Debugf(format string, v ...interface{}) {
	p.logger.Debugf(p.indexstr+format, v...)
}

// Info implements InfoLogger.Info
func (p *indexLogger) Info(msg string) {
	p.logger.Info(p.indexstr + msg)
}

// Infov implements InfoLogger.Infov
func (p *indexLogger) Infof(format string, v ...interface{}) {
	p.logger.Infof(p.indexstr+format, v...)
}

// Warn implements Logger.Warn
func (p *indexLogger) Warn(msg string) {
	p.logger.Warn(p.indexstr + msg)
}

// Warnf implements Logger.Warnf
func (p *indexLogger) Warnf(format string, v ...interface{}) {
	p.logger.Warnf(p.indexstr+format, v...)
}
