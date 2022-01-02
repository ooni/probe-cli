package scrubber

import (
	"fmt"
	"testing"
)

type savingLogger struct {
	debug []string
	info  []string
	warn  []string
}

func (sl *savingLogger) Debug(message string) {
	sl.debug = append(sl.debug, message)
}

func (sl *savingLogger) Debugf(format string, v ...interface{}) {
	sl.Debug(fmt.Sprintf(format, v...))
}

func (sl *savingLogger) Info(message string) {
	sl.info = append(sl.info, message)
}

func (sl *savingLogger) Infof(format string, v ...interface{}) {
	sl.Info(fmt.Sprintf(format, v...))
}

func (sl *savingLogger) Warn(message string) {
	sl.warn = append(sl.warn, message)
}

func (sl *savingLogger) Warnf(format string, v ...interface{}) {
	sl.Warn(fmt.Sprintf(format, v...))
}

func TestScrubLogger(t *testing.T) {
	input := "failure: 130.192.91.211:443: no route the host"
	expect := "failure: [scrubbed]: no route the host"

	t.Run("for debug", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Debug(input)
		if len(logger.debug) != 1 && len(logger.info) != 0 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.debug[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for debugf", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Debugf("%s", input)
		if len(logger.debug) != 1 && len(logger.info) != 0 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.debug[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for info", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Info(input)
		if len(logger.debug) != 0 && len(logger.info) != 1 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.info[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for infof", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Infof("%s", input)
		if len(logger.debug) != 0 && len(logger.info) != 1 && len(logger.warn) != 0 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.info[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for warn", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Warn(input)
		if len(logger.debug) != 0 && len(logger.info) != 0 && len(logger.warn) != 1 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.warn[0] != expect {
			t.Fatal("unexpected output written")
		}
	})

	t.Run("for warnf", func(t *testing.T) {
		logger := new(savingLogger)
		scrubber := &Logger{Logger: logger}
		scrubber.Warnf("%s", input)
		if len(logger.debug) != 0 && len(logger.info) != 0 && len(logger.warn) != 1 {
			t.Fatal("unexpected number of log lines written")
		}
		if logger.warn[0] != expect {
			t.Fatal("unexpected output written")
		}
	})
}
