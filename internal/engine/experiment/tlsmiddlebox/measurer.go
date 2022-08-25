package tlsmiddlebox

//
// Measurer
//

import (
	"context"
	"errors"
	"fmt"
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

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be tlshandshake or https")
)

// // Run implements ExperimentMeasurer.Run.
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
	scheme := parsed.Scheme
	if scheme != "tlshandshake" && scheme != "https" {
		return errInvalidScheme
	}
	tk := NewTestKeys()
	measurement.TestKeys = tk
	sni := m.config.sni(parsed.Host)
	wg := new(sync.WaitGroup)
	// 1. perform a DNSLookup
	addrs, err := m.DNSLookup(ctx, 0, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Hostname(), tk)
	if err != nil {
		return err
	}
	// 2. measure addresses
	addrs = prepareAddrs(addrs, parsed.Port())
	for i, addr := range addrs {
		wg.Add(1)
		go m.TraceAddress(ctx, int64(i), measurement.MeasurementStartTimeSaved, sess.Logger(), addr, sni, tk, wg)
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
	tk.addTrace([]*CompleteTrace{trace})
	err := m.TCPConnect(ctx, index, zeroTime, logger, address, tk)
	if err != nil {
		return err
	}
	m.TLSTrace(ctx, index, zeroTime, logger, address, sni, trace)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) *Measurer {
	return &Measurer{config: config}
}
