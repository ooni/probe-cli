package webconnectivity

//
// Measurer
//

import (
	"context"
	"errors"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Measurer for the web_connectivity experiment.
type Measurer struct {
	// Contains the experiment's config.
	Config *Config
}

// NewExperimentMeasurer creates a new model.ExperimentMeasurer.
func NewExperimentMeasurer(config *Config) model.ExperimentMeasurer {
	return &Measurer{
		Config: config,
	}
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string {
	return "web_connectivity"
}

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string {
	return "0.5.0"
}

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks) error {
	// Reminder: when this function returns an error, the measurement result
	// WILL NOT be submitted to the OONI backend. You SHOULD only return an error
	// for fundamental errors (e.g., the input is invalid or missing).

	// honour InputOrQueryBackend
	input := measurement.Input
	if input == "" {
		return errors.New("no input provided")
	}

	// convert the input string to a URL
	inputParser := &InputParser{
		AcceptedSchemes: []string{
			"http",
			"https",
		},
		AllowEndpoints: false,
		DefaultScheme:  "",
	}
	URL, err := inputParser.Parse(string(measurement.Input))
	if err != nil {
		return err
	}

	// initialize the experiment's test keys
	tk := NewTestKeys()
	measurement.TestKeys = tk

	// create variables required to run parallel tasks
	idGenerator := &atomicx.Int64{}
	wg := &sync.WaitGroup{}

	// start background tasks
	resos := &DNSResolvers{
		Domain:          URL.Hostname(),
		IDGenerator:     idGenerator,
		Logger:          sess.Logger(),
		TestKeys:        tk,
		URL:             URL,
		ZeroTime:        measurement.MeasurementStartTimeSaved,
		WaitGroup:       wg,
		DNSOverHTTPSURL: "",
		UDPAddress:      "",
	}
	resos.Start(ctx)

	// wait for background tasks to join
	wg.Wait()

	// If the context passed to us has been cancelled, we cannot
	// trust this experiment's results to be okay.
	if err := ctx.Err(); err != nil {
		return err
	}

	// perform any deferred computation on the test keys
	tk.finalize()

	// return whether there was a fundamental failure, which would prevent
	// the measurement from being submitted to the OONI collector.
	return tk.fundamentalFailure
}
