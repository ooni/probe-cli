package nettests

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/context"
)

// ExperimentFactory constructs a generic [Experiment].
type ExperimentFactory interface {
	// NewExperiment creates a new instance of [Experiment].
	NewExperiment(callbacks model.ExperimentCallbacks) Experiment
}

// Experiment is the generic API of all experiments.
type Experiment interface {
	// GetSummaryKeys returns a data structure containing a
	// summary of the test keys for ooniprobe.
	GetSummaryKeys(m *model.Measurement) (any, error)

	// KibiBytesReceived accounts for the KibiBytes received by the experiment.
	KibiBytesReceived() float64

	// KibiBytesSent is like KibiBytesReceived but for the bytes sent.
	KibiBytesSent() float64

	// Measure performs a synchronous measurement and returns
	// either a valid measurement or an error.
	Measure(ctx context.Context, input string) (*model.Measurement, error)

	// Name returns the experiment name.
	Name() string

	// ReportID returns the reportID used by this experiment.
	ReportID() string
}

// ErrMissingCheckInConfig is the error returned when we do not have
// a check-in configuration for the selected experiment.
var ErrMissingCheckInConfig = errors.New("nettests: missing check-in config")
