package mocks

// Logger allows mocking a logger.
type Logger struct {
	MockDebug func(message string)

	MockDebugf func(format string, v ...interface{})
}

// Debug calls MockDebug.
func (lo *Logger) Debug(message string) {
	lo.MockDebug(message)
}

// Debugf calls MockDebugf.
func (lo *Logger) Debugf(format string, v ...interface{}) {
	lo.MockDebugf(format, v...)
}
