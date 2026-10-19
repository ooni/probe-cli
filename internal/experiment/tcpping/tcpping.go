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

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "tcpping"
	testVersion = "0.2.1"
)

// Config contains the experiment configuration.
type Config struct {
	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `json:"delay,omitempty" ooni:"number of milliseconds to wait before sending each ping"`

	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `json:"repetitions,omitempty" ooni:"number of times to repeat the measurement"`
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
type Measurer struct{}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType

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
		return errNoInputProvided
	}
	parsed, err := url.Parse(input)
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
	go m.tcpPingLoop(ctx, config, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, out)
	for len(tk.Pings) < int(config.repetitions()) {
		tk.Pings = append(tk.Pings, <-out)
	}
	return nil // return nil so we always submit the measurement
}

// tcpPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tcpPingLoop(ctx context.Context, config *Config, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *SinglePing) {
	ticker := time.NewTicker(config.delay())
	defer ticker.Stop()
	for i := int64(0); i < config.repetitions(); i++ {
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
	ol := logx.NewOperationLogger(logger, "TCPPing #%d %s", index, address)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	_ = measurexlite.MaybeClose(conn)
	sp := &SinglePing{
		TCPConnect: trace.FirstTCPConnectOrNil(), // record the first connect from the buffer
	}
	return sp
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}
