package echcheck

import (
	"context"
	"errors"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"net"
	"net/url"
	"time"
)

const (
	testName      = "echcheck"
	testVersion   = "0.1.0"
	defaultDomain = "https://example.org"
)

var (
	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidInputScheme indicates that the input scheme is invalid
	errInvalidInputScheme = errors.New("input scheme must be https")
)

// TestKeys contains echcheck test keys.
type TestKeys struct {
	Control model.ArchivalTLSOrQUICHandshakeResult `json:"control"`
	Target  model.ArchivalTLSOrQUICHandshakeResult `json:"target"`
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
func (m *Measurer) Run(
	ctx context.Context,
	args *model.ExperimentArgs,
) error {
	if args.Measurement.Input == "" {
		args.Measurement.Input = defaultDomain
	}
	parsed, err := url.Parse(string(args.Measurement.Input))
	if err != nil {
		return errInputIsNotAnURL
	}
	if parsed.Scheme != "https" {
		return errInvalidInputScheme
	}

	// 1. perform a DNSLookup
	trace := measurexlite.NewTrace(0, args.Measurement.MeasurementStartTimeSaved)
	resolver := trace.NewParallelDNSOverHTTPSResolver(args.Session.Logger(), m.config.resolverURL())
	addrs, err := resolver.LookupHost(ctx, parsed.Host)
	if err != nil {
		return err
	}
	runtimex.Assert(len(addrs) > 0, "expected at least one entry in addrs")
	address := net.JoinHostPort(addrs[0], "443")

	// 2. Set up TCP connections
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	conn2, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	// 3. Conduct and measure control and target TLS handshakes in parallel
	controlChannel := make(chan model.ArchivalTLSOrQUICHandshakeResult)
	targetChannel := make(chan model.ArchivalTLSOrQUICHandshakeResult)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		controlChannel <- *handshake(ctx, conn, args.Measurement.MeasurementStartTimeSaved, address, parsed.Host)
	}()

	go func() {
		targetChannel <- *handshakeWithEch(ctx, conn2, args.Measurement.MeasurementStartTimeSaved, address, parsed.Host)
	}()

	control := <-controlChannel
	target := <-targetChannel

	args.Measurement.TestKeys = TestKeys{Control: control, Target: target}

	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
