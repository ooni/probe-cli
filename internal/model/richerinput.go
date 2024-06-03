package model

//
// Richer input
//
// See XXX
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/erroror"
)

// RicherInputSession is the richer inputs's view of a session.
type RicherInputSession interface {
	// A RicherInputSession is also an [ExperimentSession].
	ExperimentSession

	// CheckIn invokes the check-in API.
	CheckIn(ctx context.Context, req *OOAPICheckInConfig) (*OOAPICheckInResult, error)

	// OpenReport opens a new report.
	OpenReport(ctx context.Context, tmpl *OOAPIReportTemplate) (OOAPIReport, error)

	// Platform returns the platform (e.g., "linux").
	Platform() string

	// ProbeASNString returns the probe ASN as a string.
	ProbeASNString() string

	// ProbeIP returns the probe IP addr.
	ProbeIP() string

	// ProbeNetworkName returns the probe ASN network name.
	ProbeNetworkName() string

	// ResolverASNString returns the resolver ASN as a string.
	ResolverASNString() string

	// ResolverNetworkName returns the resoler ASN network name.
	ResolverNetworkName() string

	// SoftwareName returns the software name.
	SoftwareName() string

	// SoftwareVersion returns the software version.
	SoftwareVersion() string
}

const (
	// DefaultCategoryCode is the category code that one should return in the [RicherInputTarget]
	// CategoryCode method when the category code is not known.
	DefaultCategoryCode = "MISC"

	// DefaultCountryCode is the category code that one should return in the [RicherInputTarget]
	// CountryCode method when the country code is not known.
	DefaultCountryCode = "ZZ"
)

// RicherInputTarget is a generic richer-input measurement target.
type RicherInputTarget interface {
	// CategoryCode returns the category code in the citizenlab test-lists.
	//
	// Use DefaultCategoryCode when you don't know which category code to return.
	CategoryCode() string

	// CountryCode returns the country code in the citizenlab test-lists.
	//
	// Use DefaultCountryCode when you don't know the country code to return.
	CountryCode() string

	// Input returns the input to save into the measurement.
	//
	// This method SHOULD typically return a URL.
	Input() MeasurementInput

	// Options returns the options to save into the measurement.
	//
	// This function MUST avoid including into the returned list all the options:
	//
	// 1. whose Go name starts with "Safe";
	//
	// 2. whose corresponding JSON name starts with "safe_";
	//
	// 3. whose value is the Go zero value for the type.
	Options() []string
}

// RicherInputCallbacks contains callbacks invoked by richer-input-aware experiments.
//
// Implementations of this type MUST be goroutine safe since callbacks will
// always be invoked from background goroutines.
type RicherInputCallbacks interface {
	// OnProgress provides information about an experiment progress.
	//
	// The progress argument is a value between 0.0 and 1.0.
	OnProgress(progress float64, message string)

	// OnTargets is invoked with the list of targets that we will be using, which
	// is typically obtained by calling experiment-specific backend APIs.
	//
	// We provide all the targets in a single call such that inserting them into
	// sqlite3 happens via a single SQL transaction.
	OnTargets(target []RicherInputTarget)
}

// RicherInputConfig contains the richer input config passed to an experiment.
type RicherInputConfig struct {
	// Annotations contains OPTIONAL annotations for the experiment.
	Annotations map[string]string

	// Callbacks contains MANDATORY richer-input callbacks.
	//
	// Note: the provided RicherInputCallbacks implementation MUST be such
	// that every method does not cause any data race.
	Callbacks RicherInputCallbacks

	// ExtraOptions contains OPTIONAL extra options for the experiment
	// provided from the command line or OONI Run v2. This field's type
	// is flexible in that each option could either be a string or the
	// correct field type (assuming scalar fields).
	//
	// TODO(bassosimone): we should find a cleaner solution to this problem
	// that also allows setting non-scalar fields from OONI Run v2.
	ExtraOptions map[string]any

	// Inputs contains the OPTIONAL experiment inputs.
	Inputs []string

	// InputFilePaths contains OPTIONAL files to read inputs from.
	InputFilePaths []string

	// MaxRuntime is the OPTIONAL maximum runtime in seconds. This field is only
	// effective when we're measuring two or more inputs.
	MaxRuntime int64

	// RandomizeInputs OPTIONALLY indicates we should randomize inputs.
	RandomizeInputs bool
}

// ContainsUserConfiguredInput returns true iff c contains any user-configured input.
func (c *RicherInputConfig) ContainsUserConfiguredInput() bool {
	return len(c.ExtraOptions) > 0 || len(c.Inputs) > 0 || len(c.InputFilePaths) > 0
}

// RicherInputExperiment is a richer-input-aware experiment.
type RicherInputExperiment interface {
	// KibiBytesReceived accounts for the KibiBytes received by the experiment.
	KibiBytesReceived() float64

	// KibiBytesSent is like KibiBytesReceived but for the bytes sent.
	KibiBytesSent() float64

	// Name returns the experiment name.
	Name() string

	// OpenReport opens a report for this experiment.
	OpenReport(ctx context.Context) error

	// ReportID returns the open report's ID, if we have opened a report
	// successfully before, or an empty string, otherwise.
	ReportID() string

	// Start runs the measurement using the given configuration in a
	// background goroutine. The specific algorithm depend on configuration:
	//
	// 1. If config does not contain any user configuted input, we will call
	// an experiment-specific probe-services APIs to retrieve richer input, or we
	// will use static defaults hardcoded inside the experiment implementation.
	//
	// 2. Otherwise, the experiment will use the provided config.
	//
	// The background goroutine will emit events on the returned channel and
	// close the channel when it's done running.
	Start(ctx context.Context, config *RicherInputConfig) <-chan *erroror.Value[*Measurement]

	// SubmitMeasurement submits a measurement and updates its
	// report ID field on successful submission.
	SubmitMeasurement(ctx context.Context, m *Measurement) error
}
