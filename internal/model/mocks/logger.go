package mocks

// Logger allows mocking a logger.
type Logger struct {
	MockDebug  func(message string)
	MockDebugf func(format string, v ...interface{})
	MockInfo   func(message string)
	MockInfof  func(format string, v ...interface{})
	MockWarn   func(message string)
	MockWarnf  func(format string, v ...interface{})
}

// Debug calls MockDebug.
func (lo *Logger) Debug(message string) {
	lo.MockDebug(message)
}

// Debugf calls MockDebugf.
func (lo *Logger) Debugf(format string, v ...interface{}) {
	lo.MockDebugf(format, v...)
}

// Info calls MockInfo.
func (lo *Logger) Info(message string) {
	lo.MockInfo(message)
}

// Infof calls MockInfof.
func (lo *Logger) Infof(format string, v ...interface{}) {
	lo.MockInfof(format, v...)
}

// Warn calls MockWarn.
func (lo *Logger) Warn(message string) {
	lo.MockWarn(message)
}

// Warnf calls MockWarnf.
func (lo *Logger) Warnf(format string, v ...interface{}) {
	lo.MockWarnf(format, v...)
}
