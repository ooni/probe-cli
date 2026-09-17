// Package httphostheader contains the HTTP host header network experiment.
//
// This experiment has not been specified yet. It is nonetheless available for testing
// and as a building block that other experiments could reuse.
package httphostheader

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "http_host_header"
	testVersion = "0.3.1"
)

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType
)

// Config contains the experiment config.
type Config struct {
	// TestHelperURL is the address of the test helper.
	TestHelperURL string `json:"test_helper_url,omitempty"`
}

// TestKeys contains httphost test keys.
type TestKeys struct {
	urlgetter.TestKeys
	THAddress string `json:"th_address"`
}

// Measurer performs the measurement.
type Measurer struct{}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// obtain the richer-input target
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}
	config, input := target.Config, target.URL

	if input == "" {
		return errors.New("experiment requires input")
	}
	if config.TestHelperURL == "" {
		config.TestHelperURL = "http://www.example.org"
	}
	urlgetter.RegisterExtensions(measurement)
	g := urlgetter.Getter{
		Begin: measurement.MeasurementStartTimeSaved,
		Config: urlgetter.Config{
			HTTPHost: input,
		},
		Session: sess,
		Target:  config.TestHelperURL,
	}
	tk, _ := g.Get(ctx)
	measurement.TestKeys = &TestKeys{
		TestKeys:  tk,
		THAddress: config.TestHelperURL,
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}
