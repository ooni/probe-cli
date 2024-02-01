package logx

import "github.com/ooni/probe-cli/v3/internal/logmodel"

// PrefixLogger is a logger with a prefix.
type PrefixLogger struct {
	Prefix string
	Logger logmodel.Logger
}

var _ logmodel.Logger = &PrefixLogger{}

// Debug implements DebugLogger.Debug
func (p *PrefixLogger) Debug(msg string) {
	p.Logger.Debug(p.Prefix + msg)
}

// Debugf implements DebugLogger.Debugf
func (p *PrefixLogger) Debugf(format string, v ...interface{}) {
	p.Logger.Debugf(p.Prefix+format, v...)
}

// Info implements InfoLogger.Info
func (p *PrefixLogger) Info(msg string) {
	p.Logger.Info(p.Prefix + msg)
}

// Infov implements InfoLogger.Infov
func (p *PrefixLogger) Infof(format string, v ...interface{}) {
	p.Logger.Infof(p.Prefix+format, v...)
}

// Warn implements Logger.Warn
func (p *PrefixLogger) Warn(msg string) {
	p.Logger.Warn(p.Prefix + msg)
}

// Warnf implements Logger.Warnf
func (p *PrefixLogger) Warnf(format string, v ...interface{}) {
	p.Logger.Warnf(p.Prefix+format, v...)
}
