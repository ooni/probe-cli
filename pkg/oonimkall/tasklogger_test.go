package oonimkall

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

//
// This file contains tests for the taskLogger type.
//

func TestTaskLogger(t *testing.T) {
	// debugMessage is the debug message we expect to see.
	debugMessage := "debug message"

	// infoMessage is the info message we expect to see.
	infoMessage := "info message"

	// warningMessage is the warning message we expect to see.
	warningMessage := "warning message"

	// emitMessages is an helper function for implementing this test.
	emitMessages := func(logger model.Logger) {
		logger.Debug(debugMessage)
		logger.Debugf("%s", debugMessage)
		logger.Info(infoMessage)
		logger.Infof("%s", infoMessage)
		logger.Warn(warningMessage)
		logger.Warnf("%s", warningMessage)
	}

	// convertEventsToLogEvents converts the generic events to
	// logEvents and fails if this operation is not possible.
	convertEventsToLogEvents := func(t *testing.T, in []*event) (out []eventLog) {
		for _, ev := range in {
			if ev.Key != eventTypeLog {
				t.Fatalf("expected log event, found %s", ev.Key)
			}
			out = append(out, ev.Value.(eventLog))
		}
		return
	}

	// checkNumberOfEvents ensures we've the right number of events.
	checkNumberOfEvents := func(t *testing.T, events []eventLog, expect int) {
		if len(events) != expect {
			t.Fatalf(
				"invalid number of log events %d (expected %d)",
				len(events), expect,
			)
		}
	}

	// matchEvent ensures the given event has the right level and message.
	matchEvent := func(t *testing.T, event eventLog, level, msg string) {
		if event.LogLevel != level {
			t.Fatalf(
				"invalid log level %s (expected %s)",
				event.LogLevel, level,
			)
		}
		if event.Message != msg {
			t.Fatalf(
				"invalid log message '%s' (expected '%s')",
				event.Message, msg,
			)
		}
	}

	t.Run("debug logger", func(t *testing.T) {
		emitter := &CollectorTaskEmitter{}
		logger := newTaskLogger(emitter, logLevelDebug)
		emitMessages(logger)
		logEvents := convertEventsToLogEvents(t, emitter.Collect())
		checkNumberOfEvents(t, logEvents, 6)
		matchEvent(t, logEvents[0], logLevelDebug, debugMessage)
		matchEvent(t, logEvents[1], logLevelDebug, debugMessage)
		matchEvent(t, logEvents[2], logLevelInfo, infoMessage)
		matchEvent(t, logEvents[3], logLevelInfo, infoMessage)
		matchEvent(t, logEvents[4], logLevelWarning, warningMessage)
		matchEvent(t, logEvents[5], logLevelWarning, warningMessage)
	})

	t.Run("info logger", func(t *testing.T) {
		emitter := &CollectorTaskEmitter{}
		logger := newTaskLogger(emitter, logLevelInfo)
		emitMessages(logger)
		logEvents := convertEventsToLogEvents(t, emitter.Collect())
		checkNumberOfEvents(t, logEvents, 4)
		matchEvent(t, logEvents[0], logLevelInfo, infoMessage)
		matchEvent(t, logEvents[1], logLevelInfo, infoMessage)
		matchEvent(t, logEvents[2], logLevelWarning, warningMessage)
		matchEvent(t, logEvents[3], logLevelWarning, warningMessage)
	})

	t.Run("warn logger", func(t *testing.T) {
		emitter := &CollectorTaskEmitter{}
		logger := newTaskLogger(emitter, logLevelWarning)
		emitMessages(logger)
		logEvents := convertEventsToLogEvents(t, emitter.Collect())
		checkNumberOfEvents(t, logEvents, 2)
		matchEvent(t, logEvents[0], logLevelWarning, warningMessage)
		matchEvent(t, logEvents[1], logLevelWarning, warningMessage)
	})
}
