package portfiltering

//
// Measurer for the port-filtering experiment
//

import (
	"context"
	"errors"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "portfiltering"
	testVersion = "0.1.0"
)

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

var (
	// errInputIsNotAnURL indicates that input is not an URL
	errInvalidTestHelper = errors.New("testhelper is not an URL")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	// TODO(DecFox): Replace the localhost deployment with an OONI testhelper
	// Ensure that we only do this once we have a deployed testhelper
	testhelper := "http://127.0.0.1"
	parsed, err := url.Parse(testhelper)
	if err != nil {
		return errInvalidTestHelper
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	out := make(chan *model.ArchivalTCPConnectResult)
	go m.tcpPingLoop(ctx, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, out)
	for len(tk.TCPConnect) < len(Ports) {
		tk.TCPConnect = append(tk.TCPConnect, <-out)
	}
	return nil // return nil so we always submit the measurement
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}
