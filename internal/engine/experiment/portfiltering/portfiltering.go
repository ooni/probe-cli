// Package portscan is the experimental portscan experiment

package portfiltering

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "portfiltering"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `ooni:"number of milliseconds to wait before knocking each port"`
}

func (c *Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return time.Second
}

// TestKeys contains the experiment results.
type TestKeys struct {
	TCPConnect []*model.ArchivalTCPConnectResult `json:"tcp_connect"`
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
	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	// TODO(DecFox): Ensure that we have a deployed test-helper to use the testhelper below
	// testhelpers, _ := sess.GetTestHelpersByName("port-filtering")
	// var testhelper *model.OOAPIService
	// for _, th := range testhelpers {
	// if th.Type == "tcp" {
	// testhelper = &th
	// break
	// }
	// }
	// if testhelper == nil {
	// return ErrNoAvailableTestHelpers
	// }
	// measurement.TestHelpers = map[string]interface{}{
	// "backend": testhelper,
	// }
	testhelper := "http://localhost"
	parsed, err := url.Parse(testhelper)
	if err != nil {
		return errInputIsNotAnURL
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

// tcpPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tcpPingLoop(ctx context.Context, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *model.ArchivalTCPConnectResult) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i, port := range Ports {
		addr := net.JoinHostPort(address, port)
		go m.tcpPingAsync(ctx, int64(i), zeroTime, logger, addr, out)
		<-ticker.C
	}
}

// tcpPingAsync performs a TCP ping and emits the result onto the out channel.
func (m *Measurer) tcpPingAsync(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string, out chan<- *model.ArchivalTCPConnectResult) {
	out <- m.tcpConnect(ctx, index, zeroTime, logger, address)
}

// tcpConnect performs a TCP connect and returns the result to the caller.
func (m *Measurer) tcpConnect(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string) (out *model.ArchivalTCPConnectResult) {
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "TCPConnect #%d %s", index, address)
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	out = trace.FirstTCPConnectOrNil()
	return
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
