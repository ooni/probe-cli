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
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "tlsmiddlebox"
	testVersion = "0.1.3"
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
	// ErrNoInput indicates that no input was provided
	ErrNoInput = errors.New("no input provided")

	// ErrInputIsNotAnURL indicates that the input is not an URL.
	ErrInputIsNotAnURL = errors.New("input is not an URL")

	// ErrUnsupportedInput indicates that the input URL scheme is unsupported.
	ErrUnsupportedInput = errors.New("unsupported input scheme, input scheme must be tlstrace")

	// errInvalidTestHelper indicates that the testhelper is invalid
	errInvalidTestHelper = errors.New("invalid testhelper")

	// errInvalidTHScheme indicates that the TH scheme is invalid
	errInvalidTHScheme = errors.New("th scheme must be tlshandshake")

	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType
)

// // Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	// obtain the richer-input target
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	config := target.Config

	if !ok {
		return ErrInvalidInputType
	}

	input := target.URL

	if input == "" {
		return ErrNoInput
	}

	URL, err := url.Parse(input)

	if err != nil {
		return ErrInputIsNotAnURL
	}
	if URL.Scheme != "tlstrace" {
		return ErrUnsupportedInput
	}
	th, err := target.Config.testhelper(URL.Host)
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
		go m.TraceAddress(ctx, int64(i), measurement.MeasurementStartTimeSaved, sess.Logger(), addr, URL.Hostname(), tk, wg, config)
	}
	wg.Wait()
	return nil
}

// TraceAddress measures a single address after the DNSLookup
func (m *Measurer) TraceAddress(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, tk *TestKeys, wg *sync.WaitGroup, config *Config) error {
	defer wg.Done()
	trace := &CompleteTrace{
		Address: address,
	}
	tk.addTrace(trace)
	err := m.TCPConnect(ctx, index, zeroTime, logger, address, tk)
	if err != nil {
		return err // skip tracing if we cannot connect with default TTL
	}
	m.TLSTrace(ctx, index, zeroTime, logger, address, sni, trace, config)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() *Measurer {
	return &Measurer{}
}
