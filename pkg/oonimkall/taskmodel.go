package oonimkall

import (
	"context"
	"io"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
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

const taskABIVersion = 1

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

// type of emitted events.
const (
	eventTypeFailureIPLookup              = "failure.ip_lookup"
	eventTypeFailureASNLookup             = "failure.asn_lookup"
	eventTypeFailureCCLookup              = "failure.cc_lookup"
	eventTypeFailureMeasurement           = "failure.measurement"
	eventTypeFailureMeasurementSubmission = "failure.measurement_submission"
	eventTypeFailureReportCreate          = "failure.report_create"
	eventTypeFailureResolverLookup        = "failure.resolver_lookup"
	eventTypeFailureStartup               = "failure.startup"
	eventTypeLog                          = "log"
	eventTypeMeasurement                  = "measurement"
	eventTypeStatusEnd                    = "status.end"
	eventTypeStatusGeoIPLookup            = "status.geoip_lookup"
	eventTypeStatusMeasurementDone        = "status.measurement_done"
	eventTypeStatusMeasurementStart       = "status.measurement_start"
	eventTypeStatusMeasurementSubmission  = "status.measurement_submission"
	eventTypeStatusProgress               = "status.progress"
	eventTypeStatusQueued                 = "status.queued"
	eventTypeStatusReportCreate           = "status.report_create"
	eventTypeStatusResolverLookup         = "status.resolver_lookup"
	eventTypeStatusStarted                = "status.started"
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

type eventEmpty struct{}

// eventFailure contains information on a failure.
type eventFailure struct {
	Failure string `json:"failure"`
}

// eventLog is an event containing a log message.
type eventLog struct {
	LogLevel string `json:"log_level"`
	Message  string `json:"message"`
}

type eventMeasurementGeneric struct {
	Failure string `json:"failure,omitempty"`
	Idx     int64  `json:"idx"`
	Input   string `json:"input"`
	JSONStr string `json:"json_str,omitempty"`
}

type eventStatusEnd struct {
	DownloadedKB float64 `json:"downloaded_kb"`
	Failure      string  `json:"failure"`
	UploadedKB   float64 `json:"uploaded_kb"`
}

type eventStatusGeoIPLookup struct {
	ProbeASN         string `json:"probe_asn"`
	ProbeCC          string `json:"probe_cc"`
	ProbeIP          string `json:"probe_ip"`
	ProbeNetworkName string `json:"probe_network_name"`
}

// eventStatusProgress reports progress information.
type eventStatusProgress struct {
	Message    string  `json:"message"`
	Percentage float64 `json:"percentage"`
}

type eventStatusReportGeneric struct {
	ReportID string `json:"report_id"`
}

type eventStatusResolverLookup struct {
	ResolverASN         string `json:"resolver_asn"`
	ResolverIP          string `json:"resolver_ip"`
	ResolverNetworkName string `json:"resolver_network_name"`
}

// event is an event emitted by a task. This structure extends the event
// described by MK v0.10.9 FFI API (https://git.io/Jv4Rv).
type event struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
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

// taskKVStoreFSBuilder constructs a KVStore with
// filesystem backing for running tests.
type taskKVStoreFSBuilder interface {
	// NewFS creates a new KVStore using the filesystem.
	NewFS(path string) (model.KeyValueStore, error)
}

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

//
// Task Running
//
// This section contains the interfaces allowing us
// to run a task until completion.
//

// taskRunner runs a task until completion.
type taskRunner interface {
	// Run runs until completion.
	Run(ctx context.Context)
}

//
// Task Settings
//
// This section defines the settings used by a task.
//

// Settings contains settings for a task. This structure derives from
// the one described by MK v0.10.9 FFI API (https://git.io/Jv4Rv), yet
// since 2020-12-03 we're not backwards compatible anymore.
type settings struct {
	// Annotations contains the annotations to be added
	// to every measurements performed by the task.
	Annotations map[string]string `json:"annotations,omitempty"`

	// AssetsDir is the directory where to store assets. This
	// field is an extension of MK's specification. If
	// this field is empty, the task won't start.
	AssetsDir string `json:"assets_dir"`

	// DisabledEvents contains disabled events. See
	// https://git.io/Jv4Rv for the events names.
	//
	// This setting is currently ignored. We noticed the
	// code was ignoring it on 2021-12-01.
	DisabledEvents []string `json:"disabled_events,omitempty"`

	// Inputs contains the inputs. The task will fail if it
	// requires input and you provide no input.
	Inputs []string `json:"inputs,omitempty"`

	// LogLevel contains the logs level. See https://git.io/Jv4Rv
	// for the names of the available log levels.
	LogLevel string `json:"log_level,omitempty"`

	// Name contains the task name. By https://git.io/Jv4Rv the
	// names are in camel case, e.g. `Ndt`.
	Name string `json:"name"`

	// Options contains the task options.
	Options settingsOptions `json:"options"`

	// Proxy allows you to optionally force a specific proxy
	// rather than using no proxy (the default).
	//
	// Use `psiphon:///` to force using Psiphon with the
	// embedded configuration file. Not all builds have
	// an embedded configuration file, but OONI builds have
	// such a file, so they can use this functionality.
	//
	// Use `socks5://10.0.0.1:9050/` to connect to a SOCKS5
	// proxy running on 10.0.0.1:9050. This could be, for
	// example, a suitably configured `tor` instance.
	Proxy string

	// StateDir is the directory where to store persistent data. This
	// field is an extension of MK's specification. If
	// this field is empty, the task won't start.
	StateDir string `json:"state_dir"`

	// TempDir is the temporary directory. This field is an extension of MK's
	// specification. If this field is empty, we will pick the tempdir that
	// ioutil.TempDir uses by default, which may not work on mobile. According
	// to our experiments as of 2020-06-10, leaving the TempDir empty works
	// for iOS and does not work for Android.
	TempDir string `json:"temp_dir"`

	// TunnelDir is the directory where to store persistent state
	// related to circumvention tunnels. This directory is required
	// only if you want to use the tunnels. Added since 3.10.0.
	TunnelDir string `json:"tunnel_dir"`

	// Version indicates the version of this structure.
	Version int64 `json:"version"`
}

// settingsOptions contains the settings options
type settingsOptions struct {
	// MaxRuntime is the maximum runtime expressed in seconds. A negative
	// value for this field disables the maximum runtime. Using
	// a zero value will also mean disabled. This is not the
	// original behaviour of Measurement Kit, which used to run
	// for zero time in such case.
	MaxRuntime float64 `json:"max_runtime,omitempty"`

	// NoCollector indicates whether to use a collector
	NoCollector bool `json:"no_collector,omitempty"`

	// ProbeServicesBaseURL contains the probe services base URL.
	ProbeServicesBaseURL string `json:"probe_services_base_url,omitempty"`

	// SoftwareName is the software name. If this option is not
	// present, then the library startup will fail.
	SoftwareName string `json:"software_name,omitempty"`

	// SoftwareVersion is the software version. If this option is not
	// present, then the library startup will fail.
	SoftwareVersion string `json:"software_version,omitempty"`
}
