package tlsmiddlebox

//
// Measurer
//

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "tlsmiddlebox"
	testVersion = "0.1.0"
)

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// errNoInputProvided indicates you didn't provide any input
	errNoInputProvided = errors.New("no input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidInputScheme indicates that the input scheme is invalid
	errInvalidInputScheme = errors.New("input scheme must be tlstrace")

	// errInvalidTestHelper indicates that the testhelper is invalid
	errInvalidTestHelper = errors.New("invalid testhelper")

	// errInvalidTHScheme indicates that the TH scheme is invalid
	errInvalidTHScheme = errors.New("th scheme must be tlshandshake")
)

// // Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	if measurement.Input == "" {
		return errNoInputProvided
	}
	parsed, err := url.Parse(string(measurement.Input))
	if err != nil {
		return errInputIsNotAnURL
	}
	if parsed.Scheme != "tlstrace" {
		return errInvalidInputScheme
	}
	th, err := m.config.testhelper(parsed.Host)
	if err != nil {
		return errInvalidTestHelper
	}
	if th.Scheme != "tlshandshake" {
		return errInvalidTHScheme
	}
	tk := NewTestKeys()
	measurement.TestKeys = tk
	wg := new(sync.WaitGroup)
	// 1. perform a DNSLookup
	addrs, err := m.DNSLookup(ctx, 0, measurement.MeasurementStartTimeSaved, sess.Logger(), th.Hostname(), tk)
	if err != nil {
		return err
	}
	// 2. measure addresses
	addrs = prepareAddrs(addrs, th.Port())
	for i, addr := range addrs {
		wg.Add(1)
		go m.TraceAddress(ctx, int64(i), measurement.MeasurementStartTimeSaved, sess.Logger(), addr, parsed.Hostname(), tk, wg)
	}
	wg.Wait()
	return nil
}

// TraceAddress measures a single address after the DNSLookup
func (m *Measurer) TraceAddress(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, tk *TestKeys, wg *sync.WaitGroup) error {
	defer wg.Done()
	trace := &CompleteTrace{
		Address: address,
	}
	tk.addTrace(trace)
	err := m.TCPConnect(ctx, index, zeroTime, logger, address, tk)
	if err != nil {
		return err // skip tracing if we cannot connect with default TTL
	}
	m.TLSTrace(ctx, index, zeroTime, logger, address, sni, trace)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) *Measurer {
	return &Measurer{config: config}
}
