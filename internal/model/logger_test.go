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

func TestErrorToStringOrOK(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		expectedResult := ErrorToStringOrOK(nil)
		if expectedResult != "ok" {
			t.Fatal("expected ok")
		}
	})

	t.Run("on failure", func(t *testing.T) {
		err := io.EOF
		expectedResult := ErrorToStringOrOK(err)
		if expectedResult != err.Error() {
			t.Fatal("not the result we expected", expectedResult)
		}
	})
}

func TestValidLoggerOrDefault(t *testing.T) {
	t.Run("with nil argument", func(t *testing.T) {
		out := ValidLoggerOrDefault(nil)
		if out != DiscardLogger {
			t.Fatal("unexpected result")
		}
	})

	t.Run("with non-nil argument", func(t *testing.T) {
		in := &logDiscarder{}
		if ValidLoggerOrDefault(in) != in {
			t.Fatal("unexpected result")
		}
	})
}
