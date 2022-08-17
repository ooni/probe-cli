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

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "tcpping"
	testVersion = "0.2.0"
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
	TCPConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`
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
	out := make(chan *SinglePing)
	go m.tcpPingLoop(ctx, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, out)
	for len(tk.Pings) < int(m.config.repetitions()) {
		tk.Pings = append(tk.Pings, <-out)
	}
	return nil // return nil so we always submit the measurement
}

// tcpPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tcpPingLoop(ctx context.Context, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *SinglePing) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i := int64(0); i < m.config.repetitions(); i++ {
		go m.tcpPingAsync(ctx, i, zeroTime, logger, address, out)
		<-ticker.C
	}
}

// tcpPingAsync performs a TCP ping and emits the result onto the out channel.
func (m *Measurer) tcpPingAsync(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string, out chan<- *SinglePing) {
	out <- m.tcpConnect(ctx, index, zeroTime, logger, address)
}

// tcpConnect performs a TCP connect and returns the result to the caller.
func (m *Measurer) tcpConnect(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string) *SinglePing {
	// TODO(bassosimone): make the timeout user-configurable
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	trace := measurexlite.NewTrace(index, zeroTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	ol := measurexlite.NewOperationLogger(logger, "TCPPing #%d %s", index, address)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	sp := &SinglePing{
		TCPConnect: trace.TCPConnects()[0], // record the first connect from the buffer
	}
	return sp
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
