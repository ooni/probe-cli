package sessionresolver

// KVStore is a generic key-value store. We use it to store
// on disk persistent state used by this package.
type KVStore interface {
	// Get gets the value for the given key.
	Get(key string) ([]byte, error)

	// Set sets the value of the given key.
	Set(key string, value []byte) error
}

// Logger defines the common logger interface.
type Logger interface {
	// Debug emits a debug message.
	Debug(msg string)

	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Info emits an informational message.
	Info(msg string)

	// Infof format and emits an informational message.
	Infof(format string, v ...interface{})

	// Warn emits a warning message.
	Warn(msg string)

	// Warnf formats and emits a warning message.
	Warnf(format string, v ...interface{})
}
