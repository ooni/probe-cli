package oonimkall

import (
	"context"
	"io"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

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

// taskEmitterCloser is a closeable taskEmitter.
type taskEmitterCloser interface {
	taskEmitter
	io.Closer
}

//
// OONI Session
//
// For performing several operations, including running
// experiments, we need to create an OONI session.
//
// This section of the file defines the interface between
// our oonimkall API and the real session.
//
// The abstraction representing a OONI session is taskSession.
//

// taskSessionBuilder constructs a new Session.
type taskSessionBuilder interface {
	// NewSession creates a new taskSession.
	NewSession(ctx context.Context,
		config engine.SessionConfig) (taskSession, error)
}

// taskSession abstracts a OONI session.
type taskSession interface {
	// A session can be closed.
	io.Closer

	// NewExperimentBuilderByName creates the builder for constructing
	// a new experiment given the experiment's name.
	NewExperimentBuilderByName(name string) (taskExperimentBuilder, error)

	// MaybeLookupBackendsContext lookups the OONI backend unless
	// this operation has already been performed.
	MaybeLookupBackendsContext(ctx context.Context) error

	// MaybeLookupLocationContext lookups the probe location unless
	// this operation has already been performed.
	MaybeLookupLocationContext(ctx context.Context) error

	// ProbeIP must be called after MaybeLookupLocationContext
	// and returns the resolved probe IP.
	ProbeIP() string

	// ProbeASNString must be called after MaybeLookupLocationContext
	// and returns the resolved probe ASN as a string.
	ProbeASNString() string

	// ProbeCC must be called after MaybeLookupLocationContext
	// and returns the resolved probe country code.
	ProbeCC() string

	// ProbeNetworkName must be called after MaybeLookupLocationContext
	// and returns the resolved probe country code.
	ProbeNetworkName() string

	// ResolverANSString must be called after MaybeLookupLocationContext
	// and returns the resolved resolver's ASN as a string.
	ResolverASNString() string

	// ResolverIP must be called after MaybeLookupLocationContext
	// and returns the resolved resolver's IP.
	ResolverIP() string

	// ResolverNetworkName must be called after MaybeLookupLocationContext
	// and returns the resolved resolver's network name.
	ResolverNetworkName() string
}

// taskExperimentBuilder builds a taskExperiment.
type taskExperimentBuilder interface {
	// SetCallbacks sets the experiment callbacks.
	SetCallbacks(callbacks model.ExperimentCallbacks)

	// InputPolicy returns the experiment's input policy.
	InputPolicy() engine.InputPolicy

	// NewExperiment creates the new experiment.
	NewExperimentInstance() taskExperiment

	// Interruptible returns whether this experiment is interruptible.
	Interruptible() bool
}

// taskExperiment is a runnable experiment.
type taskExperiment interface {
	// KibiBytesReceived returns the KiB received by the experiment.
	KibiBytesReceived() float64

	// KibiBytesSent returns the KiB sent by the experiment.
	KibiBytesSent() float64

	// OpenReportContext opens a new report.
	OpenReportContext(ctx context.Context) error

	// ReportID must be called after a successful OpenReportContext
	// and returns the report ID for this measurement.
	ReportID() string

	// MeasureWithContext runs the measurement.
	MeasureWithContext(ctx context.Context, input string) (
		measurement *model.Measurement, err error)

	// SubmitAndUpdateMeasurementContext submits the measurement
	// and updates its report ID on success.
	SubmitAndUpdateMeasurementContext(
		ctx context.Context, measurement *model.Measurement) error
}
