package ptx

import "testing"

func TestCoverSilentLogger(t *testing.T) {
	// let us not be distracted by uncovered lines that can
	// easily covered, we can easily cover defaultLogger
	defaultLogger.Debugf("foo")
	defaultLogger.Infof("bar")
	defaultLogger.Warnf("baz")
}
