package model

import (
	"io"
	"testing"
)

func TestDiscardLoggerWorksAsIntended(t *testing.T) {
	logger := DiscardLogger
	logger.Debug("foo")
	logger.Debugf("%s", "foo")
	logger.Info("foo")
	logger.Infof("%s", "foo")
	logger.Warn("foo")
	logger.Warnf("%s", "foo")
}

func TestShowOkOnSuccess(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		expectedResult := ShowOkOnSuccess(nil)
		if expectedResult != "ok" {
			t.Fatal("expected ok")
		}
	})

	t.Run("on failure", func(t *testing.T) {
		err := io.EOF
		expectedResult := ShowOkOnSuccess(err)
		if expectedResult != err.Error() {
			t.Fatal("not the result we expected", expectedResult)
		}
	})
}
