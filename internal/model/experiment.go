package model

//
// Definition of experiment and types used by the
// implementation of all experiments.
//

import (
	"context"
)

// ExperimentSession is the experiment's view of a session.
type ExperimentSession interface {
	// GetTestHelpersByName returns a list of test helpers with the given name.
	GetTestHelpersByName(name string) ([]OOAPIService, bool)

	// DefaultHTTPClient returns the default HTTPClient used by the session.
	DefaultHTTPClient() HTTPClient

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

// ExperimentAsyncTestKeys is the type of test keys returned by an experiment
// when running in async fashion rather than in sync fashion.
type ExperimentAsyncTestKeys struct {
	// Extensions contains the extensions used by this experiment.
	Extensions map[string]int64

	// Input is the input this measurement refers to.
	Input MeasurementTarget

	// MeasurementRuntime is the total measurement runtime.
	MeasurementRuntime float64

	// TestHelpers contains the test helpers used in the experiment
	TestHelpers map[string]interface{}

	// TestKeys contains the actual test keys.
	TestKeys interface{}
}

// ExperimentMeasurerAsync is a measurer that can run in async fashion.
//
// Currently this functionality is optional, but we will likely
// migrate all experiments to use this functionality in 2022.
type ExperimentMeasurerAsync interface {
	// RunAsync runs the experiment in async fashion.
	//
	// Arguments:
	//
	// - ctx is the context for deadline/timeout/cancellation
	//
	// - sess is the measurement session
	//
	// - input is the input URL to measure
	//
	// - callbacks contains the experiment callbacks
	//
	// Returns either a channel where TestKeys are posted or an error.
	//
	// An error indicates that specific preconditions for running the experiment
	// are not met (e.g., the input URL is invalid).
	//
	// On success, the experiment will post on the channel each new
	// measurement until it is done and closes the channel.
	RunAsync(ctx context.Context, sess ExperimentSession, input string,
		callbacks ExperimentCallbacks) (<-chan *ExperimentAsyncTestKeys, error)
}

// ExperimentCallbacks contains experiment event-handling callbacks
type ExperimentCallbacks interface {
	// OnProgress provides information about an experiment progress.
	OnProgress(percentage float64, message string)
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

// ExperimentArgs contains the arguments passed to an experiment.
type ExperimentArgs struct {
	// Callbacks contains MANDATORY experiment callbacks.
	Callbacks ExperimentCallbacks

	// Measurement is the MANDATORY measurement in which the experiment
	// must write the results of the measurement.
	Measurement *Measurement

	// Session is the MANDATORY session the experiment can use.
	Session ExperimentSession
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
	//
	// Deprecated: new code should use a Submitter.
	ReportID() string

	// MeasureAsync runs an async measurement. This operation could post
	// one or more measurements onto the returned channel. We'll close the
	// channel when we've emitted all the measurements.
	//
	// Arguments:
	//
	// - ctx is the context for deadline/cancellation/timeout;
	//
	// - input is the input (typically a URL but it could also be
	// just an endpoint or an empty string for input-less experiments
	// such as, e.g., ndt7 and dash).
	//
	// Return value:
	//
	// - on success, channel where to post measurements (the channel
	// will be closed when done) and nil error;
	//
	// - on failure, nil channel and non-nil error.
	MeasureAsync(ctx context.Context, input string) (<-chan *Measurement, error)

	// MeasureWithContext performs a synchronous measurement.
	//
	// Return value: strictly either a non-nil measurement and
	// a nil error or a nil measurement and a non-nil error.
	//
	// CAVEAT: while this API is perfectly fine for experiments that
	// return a single measurement, it will only return the first measurement
	// when used with an asynchronous experiment.
	MeasureWithContext(ctx context.Context, input string) (measurement *Measurement, err error)

	// SaveMeasurement saves a measurement on the specified file path.
	//
	// Deprecated: new code should use a Saver.
	SaveMeasurement(measurement *Measurement, filePath string) error

	// SubmitAndUpdateMeasurementContext submits a measurement and updates the
	// fields whose value has changed as part of the submission.
	//
	// Deprecated: new code should use a Submitter.
	SubmitAndUpdateMeasurementContext(
		ctx context.Context, measurement *Measurement) error

	// OpenReportContext will open a report using the given context
	// to possibly limit the lifetime of this operation.
	//
	// Deprecated: new code should use a Submitter.
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

	// SetCallbacks sets the experiment's interactive callbacks.
	SetCallbacks(callbacks ExperimentCallbacks)

	// NewExperiment creates the experiment instance.
	NewExperiment() Experiment
}

// ExperimentOptionInfo contains info about an experiment option.
type ExperimentOptionInfo struct {
	// Doc contains the documentation.
	Doc string

	// Type contains the type.
	Type string
}

// ExperimentInputLoader loads inputs from local or remote sources.
type ExperimentInputLoader interface {
	Load(ctx context.Context) ([]OOAPIURLInfo, error)
}

// Submitter submits a measurement to the OONI collector.
type Submitter interface {
	// Submit submits the measurement and updates its
	// report ID field in case of success.
	Submit(ctx context.Context, m *Measurement) error
}

// Saver saves a measurement on some persistent storage.
type Saver interface {
	SaveMeasurement(m *Measurement) error
}

// ExperimentInputProcessor processes inputs for an experiment.
type ExperimentInputProcessor interface {
	Run(ctx context.Context) error
}
