package logx

import (
	"fmt"
	"testing"
)

// scrubberSavingLogger helps writing tests for [ScrubberLogger].
type scrubberSavingLogger struct {
	debug []string
	info  []string
	warn  []string
}

func (sl *scrubberSavingLogger) Debug(message string) {
	sl.debug = append(sl.debug, message)
}

func (sl *scrubberSavingLogger) Debugf(format string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(format, v...))
}

func (sl *scrubberSavingLogger) Info(message string) {
	sl.info = append(sl.info, message)
}

func (sl *scrubberSavingLogger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

func (sl *scrubberSavingLogger) Warn(message string) {
	sl.warn = append(sl.warn, message)
}

func (sl *scrubberSavingLogger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}

func TestScrubberLogger(t *testing.T) {
	input := "failure: 130.192.91.211:443: no route the host"
	expect := "failure: [scrubbed]: no route the host"

	t.Run("for debug", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Debug(input)
		if len(logger.debug) != 1 && len(logger.info) != 0 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.debug[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for debugf", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Debugf("%s", input)
		if len(logger.debug) != 1 && len(logger.info) != 0 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.debug[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for info", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Info(input)
		if len(logger.debug) != 0 && len(logger.info) != 1 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.info[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for infof", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Infof("%s", input)
		if len(logger.debug) != 0 && len(logger.info) != 1 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.info[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for warn", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Warn(input)
		if len(logger.debug) != 0 && len(logger.info) != 0 && len(logger.warn) != 1 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.warn[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for warnf", func(t *testing.T) {
		logger := new(scrubberSavingLogger)
		scrubber := &ScrubberLogger{Logger: logger}
		scrubber.Warnf("%s", input)
		if len(logger.debug) != 0 && len(logger.info) != 0 && len(logger.warn) != 1 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.warn[0] != expect {
			t.Fatal("unexpected output written")
		}
	})
}
