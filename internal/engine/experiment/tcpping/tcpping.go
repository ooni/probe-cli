// Package tcpping is the experimental tcpping experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-032-tcpping.md.
package tcpping

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "tcpping"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `ooni:"number of milliseconds to wait before sending each ping"`

	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`
}

func (c *Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return time.Second
}

func (c *Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Pings []*SinglePing `json:"pings"`
}

// SinglePing contains the results of a single ping.
type SinglePing struct {
	TCPConnect []*measurex.ArchivalTCPConnect `json:"tcp_connect"`
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

var (
	// errNoInputProvided indicates you didn't provide any input
	errNoInputProvided = errors.New("not input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be tcpconnect")

	// errMissingPort indicates that there is no port.
	errMissingPort = errors.New("the URL must include a port")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	if measurement.Input == "" {
		return errNoInputProvided
	}
	parsed, err := url.Parse(string(measurement.Input))
	if err != nil {
		return fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "tcpconnect" {
		return errInvalidScheme
	}
	if parsed.Port() == "" {
		return errMissingPort
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	out := make(chan *measurex.EndpointMeasurement)
	mxmx := measurex.NewMeasurerWithDefaultSettings()
	go m.tcpPingLoop(ctx, mxmx, parsed.Host, out)
	for len(tk.Pings) < int(m.config.repetitions()) {
		meas := <-out
		tk.Pings = append(tk.Pings, &SinglePing{
			TCPConnect: measurex.NewArchivalTCPConnectList(meas.Connect),
		})
	}
	return nil // return nil so we always submit the measurement
}

// tcpPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tcpPingLoop(ctx context.Context, mxmx *measurex.Measurer,
	address string, out chan<- *measurex.EndpointMeasurement) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i := int64(0); i < m.config.repetitions(); i++ {
		go m.tcpPingAsync(ctx, mxmx, address, out)
		<-ticker.C
	}
}

// tcpPingAsync performs a TCP ping and emits the result onto the out channel.
func (m *Measurer) tcpPingAsync(ctx context.Context, mxmx *measurex.Measurer,
	address string, out chan<- *measurex.EndpointMeasurement) {
	out <- m.tcpConnect(ctx, mxmx, address)
}

// tcpConnect performs a TCP connect and returns the result to the caller.
func (m *Measurer) tcpConnect(ctx context.Context, mxmx *measurex.Measurer,
	address string) *measurex.EndpointMeasurement {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return mxmx.TCPConnect(ctx, address)
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
