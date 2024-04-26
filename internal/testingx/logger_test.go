package testingx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestLogger(t *testing.T) {
	logger := &Logger{}

	logger.Debug("foobar")
	logger.Debugf("foo%s", "baz")
	expectDebug := []string{"foobar", "foobaz"}

	logger.Info("barfoo")
	logger.Infof("bar%s", "baz")
	expectInfo := []string{"barfoo", "barbaz"}

	logger.Warn("jarjar")
	logger.Warnf("jar%s", "baz")
	expectWarn := []string{"jarjar", "jarbaz"}

	if diff := cmp.Diff(expectDebug, logger.DebugLines()); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(expectInfo, logger.InfoLines()); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(expectWarn, logger.WarnLines()); diff != "" {
		t.Fatal(diff)
	}
}
