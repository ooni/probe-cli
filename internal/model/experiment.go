package model

//
// Definition of experiment and types used by the
// implementation of all experiments.
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrNoAvailableTestHelpers is emitted when there are no available test helpers.
var ErrNoAvailableTestHelpers = errors.New("no available helpers")

// ExperimentSession is the experiment's view of a session.
type ExperimentSession interface {
	// GetTestHelpersByName returns a list of test helpers with the given name.
	GetTestHelpersByName(name string) ([]OOAPIService, bool)

	// DefaultHTTPClient returns the default HTTPClient used by the session.
	DefaultHTTPClient() HTTPClient

	// FetchOpenVPNConfig returns vpn config as a serialized JSON or an error.
	FetchOpenVPNConfig(ctx context.Context, provider, cc string) (*OOAPIVPNProviderConfig, error)

	// FetchPsiphonConfig returns psiphon's config as a serialized JSON or an error.
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)

	// FetchTorTargets returns the targets for the Tor experiment or an error.
	FetchTorTargets(ctx context.Context, cc string) (map[string]OOAPITorTarget, error)

	// Logger returns the logger used by the session.
	Logger() Logger

	// ProbeCC returns the country code.
	ProbeCC() string

	// ResolverIP returns the resolver's IP.
	ResolverIP() string

	// TempDir returns the session's temporary directory.
	TempDir() string

	// TorArgs returns the arguments we should pass to tor when executing it.
	TorArgs() []string

	// TorBinary returns the path of the tor binary.
	TorBinary() string

	// TunnelDir is the directory where to store tunnel information.
	TunnelDir() string

	// UserAgent returns the user agent we should be using when we're fine
	// with identifying ourselves as ooniprobe.
	UserAgent() string
}

// ExperimentCallbacks contains experiment event-handling callbacks.
type ExperimentCallbacks interface {
	// OnProgress provides information about the experiment's progress.
	//
	// The prog field is a number between 0.0 and 1.0 representing progress, where
	// 0.0 corresponds to 0% and 1.0 corresponds to 100%.
	OnProgress(prog float64, message string)
}

// PrinterCallbacks is the default event handler
type PrinterCallbacks struct {
	Logger
}

// NewPrinterCallbacks returns a new default callback handler
func NewPrinterCallbacks(logger Logger) PrinterCallbacks {
	return PrinterCallbacks{Logger: logger}
}

// OnProgress provides information about an experiment progress.
func (d PrinterCallbacks) OnProgress(percentage float64, message string) {
	d.Logger.Infof("[%5.1f%%] %s", percentage*100, message)
}

// ExperimentTarget contains a target for the experiment to measure.
type ExperimentTarget interface {
	// Category returns the github.com/citizenlab/test-lists category
	// code for this piece of richer input.
	//
	// Return [DefaultCategoryCode] if there's no applicable category code.
	Category() string

	// Country returns the country code for this
	// piece of richer input.
	//
	// Return [DefaultCountryCode] if there's not applicable country code.
	Country() string

	// Input returns the experiment input, which is typically a URL.
	Input() string

	// Options transforms the options contained by this target
	// into a []string containing options as they were provided
	// using the command line `-O option=value` syntax.
	//
	// This method MUST NOT serialize all the options whose name
	// starts with the "Safe" prefix. This method MUST skip serializing
	// sensitive options, non-scalar options, and zero value options.
	//
	// Consider using the [experimentconfig] package to serialize.
	Options() []string

	// String MUST return the experiment input.
	//
	// Implementation note: previously existing code often times treated
	// the input as a string and, crucially, printed it using %s. To be
	// robust with respect to introducing richer input, we would like the
	// code to print in output the same value as before, which possibly
	// is processed by the desktop app. This is the reason why we are
	// introducing an explicit String() method and why we say that this
	// method MUST return the experiment input.
	String() string
}

var _ fmt.Stringer = ExperimentTarget(nil)

// ExperimentArgs contains the arguments passed to an experiment.
type ExperimentArgs struct {
	// Callbacks contains MANDATORY experiment callbacks.
	Callbacks ExperimentCallbacks

	// Measurement is the MANDATORY measurement in which the experiment
	// must write the results of the measurement.
	Measurement *Measurement

	// Session is the MANDATORY session the experiment can use.
	Session ExperimentSession

	// Target is the OPTIONAL target we're measuring.
	//
	// Only richer-input-aware experiments use this field. These experiments
	// SHOULD be defensive and handle the case where this field is nil.
	Target ExperimentTarget
}

// ExperimentMeasurer is the interface that allows to run a
// measurement for a specific experiment.
type ExperimentMeasurer interface {
	// ExperimentName returns the experiment name.
	ExperimentName() string

	// ExperimentVersion returns the experiment version.
	ExperimentVersion() string

	// Run runs the experiment with the specified context, session,
	// measurement, and experiment calbacks. This method should only
	// return an error in case the experiment could not run (e.g.,
	// a required input is missing). Otherwise, the code should just
	// set the relevant OONI error inside of the measurement and
	// return nil. This is important because the caller WILL NOT submit
	// the measurement if this method returns an error.
	Run(ctx context.Context, args *ExperimentArgs) error
}

// Experiment is an experiment instance.
type Experiment interface {
	// KibiBytesReceived accounts for the KibiBytes received by the experiment.
	KibiBytesReceived() float64

	// KibiBytesSent is like KibiBytesReceived but for the bytes sent.
	KibiBytesSent() float64

	// Name returns the experiment name.
	Name() string

	// ReportID returns the open report's ID, if we have opened a report
	// successfully before, or an empty string, otherwise.
	ReportID() string

	// MeasureWithContext measures the given experiment target.
	//
	// Return value: either a non-nil measurement and a nil error
	// or a nil measurement and a non-nil error.
	MeasureWithContext(ctx context.Context, target ExperimentTarget) (measurement *Measurement, err error)

	// SubmitAndUpdateMeasurementContext submits a measurement and updates the
	// fields whose value has changed as part of the submission.
	SubmitAndUpdateMeasurementContext(
		ctx context.Context, measurement *Measurement) (string, error)

	// OpenReportContext will open a report using the given context
	// to possibly limit the lifetime of this operation.
	OpenReportContext(ctx context.Context) error
}

// InputPolicy describes the experiment policy with respect to input. That is
// whether it requires input, optionally accepts input, does not want input.
type InputPolicy string

const (
	// InputOrQueryBackend indicates that the experiment requires
	// external input to run and that this kind of input is URLs
	// from the citizenlab/test-lists repository. If this input
	// not provided to the experiment, then the code that runs the
	// experiment is supposed to fetch from URLs from OONI's backends.
	InputOrQueryBackend = InputPolicy("or_query_backend")

	// InputStrictlyRequired indicates that the experiment
	// requires input and we currently don't have an API for
	// fetching such input. Therefore, either the user specifies
	// input or the experiment will fail for the lack of input.
	InputStrictlyRequired = InputPolicy("strictly_required")

	// InputOptional indicates that the experiment handles input,
	// if any; otherwise it fetchs input/uses a default.
	InputOptional = InputPolicy("optional")

	// InputNone indicates that the experiment does not want any
	// input and ignores the input if provided with it.
	InputNone = InputPolicy("none")

	// We gather input from StaticInput and SourceFiles. If there is
	// input, we return it. Otherwise, we return an internal static
	// list of inputs to be used with this experiment.
	InputOrStaticDefault = InputPolicy("or_static_default")
)

// ExperimentBuilder builds an experiment.
type ExperimentBuilder interface {
	// Interruptible tells you whether this is an interruptible experiment. This kind
	// of experiments (e.g. ndt7) may be interrupted mid way.
	Interruptible() bool

	// InputPolicy returns the experiment input policy.
	InputPolicy() InputPolicy

	// Options returns information about the experiment's options.
	Options() (map[string]ExperimentOptionInfo, error)

	// SetOptionAny sets an option whose value is an any value. We will use reasonable
	// heuristics to convert the any value to the proper type of the field whose name is
	// contained by the key variable. If we cannot convert the provided any value to
	// the proper type, then this function returns an error.
	SetOptionAny(key string, value any) error

	// SetOptionsAny sets options from a map[string]any. See the documentation of
	// the SetOptionAny method for more information.
	SetOptionsAny(options map[string]any) error

	// SetOptionsJSON uses the given [json.RawMessage] to initialize fields
	// of the configuration for running the experiment. The [json.RawMessage], if
	// not empty, MUST contain a serialization of the experiment config's
	// type. An empty [json.RawMessage] will silently be ignored.
	SetOptionsJSON(value json.RawMessage) error

	// SetCallbacks sets the experiment's interactive callbacks.
	SetCallbacks(callbacks ExperimentCallbacks)

	// NewExperiment creates the [Experiment] instance.
	NewExperiment() Experiment

	// NewTargetLoader creates the [ExperimentTargetLoader] instance.
	NewTargetLoader(config *ExperimentTargetLoaderConfig) ExperimentTargetLoader
}

// ExperimentTargetLoaderConfig is the configuration to create a new [ExperimentTargetLoader].
//
// The zero value is not ready to use; please, init the MANDATORY fields.
type ExperimentTargetLoaderConfig struct {
	// CheckInConfig contains OPTIONAL options for the CheckIn API. If not set, then we'll create a
	// default config. If set but there are fields inside it that are not set, then we will set them
	// to a default value.
	CheckInConfig *OOAPICheckInConfig

	// Session is the MANDATORY current measurement session.
	Session ExperimentTargetLoaderSession

	// StaticInputs contains OPTIONAL input to be added
	// to the resulting input list if possible.
	StaticInputs []string

	// SourceFiles contains OPTIONAL files to read input
	// from. Each file should contain a single input string
	// per line. We will fail if any file is unreadable
	// as well as if any file is empty.
	SourceFiles []string
}

// ExperimentTargetLoaderSession is the session according to [ExperimentTargetLoader].
type ExperimentTargetLoaderSession interface {
	// CheckIn invokes the check-in API.
	CheckIn(ctx context.Context, config *OOAPICheckInConfig) (*OOAPICheckInResult, error)

	// FetchOpenVPNConfig fetches the OpenVPN experiment configuration.
	FetchOpenVPNConfig(ctx context.Context, provider, cc string) (*OOAPIVPNProviderConfig, error)

	// Logger returns the logger to use.
	Logger() Logger

	// ProbeCC returns the probe country code.
	ProbeCC() string
}

// ExperimentOptionInfo contains info about an experiment option.
type ExperimentOptionInfo struct {
	// Doc contains the documentation.
	Doc string

	// Type contains the type.
	Type string

	// Value contains the current option value.
	Value any
}

// ExperimentTargetLoader loads targets from local or remote sources.
type ExperimentTargetLoader interface {
	Load(ctx context.Context) ([]ExperimentTarget, error)
}

// Submitter submits a measurement to the OONI collector.
type Submitter interface {
	// Submit submits the measurement and updates its
	// report ID field in case of success.
	Submit(ctx context.Context, m *Measurement) (string, error)
}

// Saver saves a measurement on some persistent storage.
type Saver interface {
	SaveMeasurement(m *Measurement) error
}

// ExperimentInputProcessor processes inputs for an experiment.
type ExperimentInputProcessor interface {
	Run(ctx context.Context) error
}
