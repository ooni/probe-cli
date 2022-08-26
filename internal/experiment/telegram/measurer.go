package telegram

//
// Measurer
//

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Measurer for the telegram experiment.
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
	return "telegram"
}

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string {
	return "0.3.0"
}

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks) error {
	// Reminder: when this function returns an error, the measurement result
	// WILL NOT be submitted to the OONI backend. You SHOULD only return an error
	// for fundamental errors (e.g., the input is invalid or missing).

	// honour InputNone
	if measurement.Input != "" {
		return errors.New("this experiment does not take any input")
	}

	// initialize the experiment's test keys
	tk := NewTestKeys()
	measurement.TestKeys = tk

	// create variables required to run parallel tasks
	idGenerator := &atomicx.Int64{}
	wg := &sync.WaitGroup{}

	// start background tasks
	systemDNSTask := &DNSResolvers{
		Domain:          webTelegramOrg,
		IDGenerator:     idGenerator,
		Logger:          sess.Logger(),
		TestKeys:        tk,
		ZeroTime:        measurement.MeasurementStartTimeSaved,
		WaitGroup:       wg,
		DNSOverHTTPSURL: "",
		UDPAddress:      "",
	}
	systemDNSTask.Start(ctx)
	for _, addr := range dataCenterAddrs {
		for _, port := range dataCenterPorts {
			dcTask := &Datacenter{
				Address:     net.JoinHostPort(addr, port),
				IDGenerator: idGenerator,
				Logger:      sess.Logger(),
				TestKeys:    tk,
				ZeroTime:    measurement.MeasurementStartTimeSaved,
				WaitGroup:   wg,
				HostHeader:  "",
				URLPath:     "",
				URLRawQuery: "",
			}
			dcTask.Start(ctx)
		}
	}

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

// dataCenterAddrs contains the data center addrs.
var dataCenterAddrs = []string{
	"149.154.175.50",
	"149.154.167.51",
	"149.154.175.100",
	"149.154.167.91",
	"149.154.171.5",
	"95.161.76.100",
}

// dataCenterPorts contains the data center ports.
var dataCenterPorts = []string{"80", "443"}
