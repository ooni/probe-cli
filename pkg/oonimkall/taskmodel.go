package oonimkall

//
// Task Model
//
// The oonimkall package allows you to run OONI network
// experiments as "tasks". This file defines all the
// underlying model entailed by running such tasks.
//
// Logging
//
// This section of the file defines the types and the
// interfaces required to implement logging.
//
// The rest of the codebase will use a generic model.Logger
// as a logger. This is a pretty fundamental interface in
// ooni/probe-cli and so it's not defined in this file.
//

// Running tasks emit logs using different log levels. We
// define log levels with the usual semantics.
//
// The logger used by a task _may_ be configured to not
// emit log events that are less severe than a given
// severity.
//
// We use the following definitions both for defining the
// log level of a log and for configuring the maximum
// acceptable log level emitted by a logger.
const (
	logLevelDebug2  = "DEBUG2"
	logLevelDebug   = "DEBUG"
	logLevelInfo    = "INFO"
	logLevelErr     = "ERR"
	logLevelWarning = "WARNING"
)

//
// Emitting Events
//
// While it is running, a task emits events. This section
// of the file defines the types needed to emit events.
//

// TODO(bassosimone): all the events inside task.go should
// eventually be migrated into the following enum.

// type of emitted events.
const (
	eventTypeLog = "log"
)

// taskEmitter is anything that allows us to
// emit events while running a task.
//
// Note that a task emitter _may_ be configured
// to ignore _some_ events though.
type taskEmitter interface {
	// Emit emits the event (unless the emitter is
	// configured to ignore this event key).
	Emit(key string, value interface{})
}
