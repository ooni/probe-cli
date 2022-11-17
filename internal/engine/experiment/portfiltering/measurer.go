package portfiltering

//
// Measurer for the port-filtering experiment
//

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"

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
	// errInputRequired indicates that no input was provided
	errInputRequired = errors.New("this experiment needs input")

	// errInvalidInput indicates an invalid port number
	errInvalidInput = errors.New("port number is invalid")

	// errInvalidTestHelper indicates that the given test helper is not an URL
	errInvalidTestHelper = errors.New("testhelper is not an URL")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	input := string(measurement.Input)
	if input == "" {
		return errInputRequired
	}
	port, err := strconv.Atoi(input)
	if err != nil || port >= 65536 || port < 0 {
		return errInvalidInput
	}
	// TODO(DecFox): Replace the localhost deployment with an OONI testhelper
	// Ensure that we only do this once we have a deployed testhelper
	th := m.config.testhelper()
	parsed, err := url.Parse(th)
	if err != nil {
		return errInvalidTestHelper
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	addr := net.JoinHostPort(parsed.Hostname(), input)
	m.tcpConnect(ctx, int64(0), measurement.MeasurementStartTimeSaved, sess.Logger(), tk, addr)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}
