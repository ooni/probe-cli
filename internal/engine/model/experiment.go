package model

import (
	"context"
	"net/http"
)

// ExperimentSession is the experiment's view of a session.
type ExperimentSession interface {
	GetTestHelpersByName(name string) ([]Service, bool)
	DefaultHTTPClient() *http.Client
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
	FetchTorTargets(ctx context.Context, cc string) (map[string]TorTarget, error)
	FetchURLList(ctx context.Context, config URLListConfig) ([]URLInfo, error)
	Logger() Logger
	ProbeCC() string
	ResolverIP() string
	TempDir() string
	TorArgs() []string
	TorBinary() string
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
	// return nil. This is important because the caller may not submit
	// the measurement if this method returns an error.
	Run(
		ctx context.Context, sess ExperimentSession,
		measurement *Measurement, callbacks ExperimentCallbacks,
	) error

	// GetSummaryKeys returns summary keys expected by ooni/probe-cli.
	GetSummaryKeys(*Measurement) (interface{}, error)
}
