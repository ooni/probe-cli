package tlsdialer

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}
