// Package httphostheader contains the HTTP host header network experiment.
//
// This experiment has not been specified yet. It is nonetheless available for testing
// and as a building block that other experiments could reuse.
package httphostheader

import (
	"context"
	"errors"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "http_host_header"
	testVersion = "0.3.0"
)

// Config contains the experiment config.
type Config struct {
	// TestHelperURL is the address of the test helper.
	TestHelperURL string
}

// TestKeys contains httphost test keys.
type TestKeys struct {
	urlgetter.TestKeys
	THAddress string `json:"th_address"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

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
	if measurement.Input == "" {
		return errors.New("experiment requires input")
	}
	if m.config.TestHelperURL == "" {
		m.config.TestHelperURL = "http://www.example.org"
	}
	urlgetter.RegisterExtensions(measurement)
	g := urlgetter.Getter{
		Begin: measurement.MeasurementStartTimeSaved,
		Config: urlgetter.Config{
			HTTPHost: string(measurement.Input),
		},
		Session: sess,
		Target:  fmt.Sprintf(m.config.TestHelperURL),
	}
	tk, _ := g.Get(ctx)
	measurement.TestKeys = &TestKeys{
		TestKeys:  tk,
		THAddress: m.config.TestHelperURL,
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
