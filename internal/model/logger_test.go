package model

import "testing"

func TestDiscardLoggerWorksAsIntended(t *testing.T) {
	logger := DiscardLogger
	logger.Debug("foo")
	logger.Debugf("%s", "foo")
	logger.Info("foo")
	logger.Infof("%s", "foo")
	logger.Warn("foo")
	logger.Warnf("%s", "foo")
}
