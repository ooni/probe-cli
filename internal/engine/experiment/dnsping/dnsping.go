// Package dnsping is the experimental dnsping experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-035-dnsping.md.
package dnsping

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "dnsping"
	testVersion = "0.2.0"
)

// Config contains the experiment configuration.
type Config struct {
	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `ooni:"number of milliseconds to wait before sending each ping"`

	// Domains is the space-separated list of domains to measure.
	Domains string `ooni:"space-separated list of domains to measure"`

	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`
}

func (c *Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return time.Second
}

func (c Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

func (c Config) domains() string {
	if c.Domains != "" {
		return c.Domains
	}
	return "edge-chat.instagram.com example.com"
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
	errInvalidScheme = errors.New("scheme must be udp")

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
	if parsed.Scheme != "udp" {
		return errInvalidScheme
	}
	if parsed.Port() == "" {
		return errMissingPort
	}
	tk := NewTestKeys()
	measurement.TestKeys = tk
	domains := strings.Split(m.config.domains(), " ")
	wg := new(sync.WaitGroup)
	wg.Add(len(domains))
	for _, domain := range domains {
		go m.dnsPingLoop(ctx, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, domain, wg, tk)
	}
	wg.Wait()
	return nil // return nil so we always submit the measurement
}

// dnsPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) dnsPingLoop(ctx context.Context, zeroTime time.Time, logger model.Logger,
	address string, domain string, wg *sync.WaitGroup, tk *TestKeys) {
	defer wg.Done()
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i := int64(0); i < m.config.repetitions(); i++ {
		wg.Add(1)
		go m.dnsRoundTrip(ctx, i, zeroTime, logger, address, domain, wg, tk)
		<-ticker.C
	}
}

// dnsRoundTrip performs a round trip and returns the results to the caller.
func (m *Measurer) dnsRoundTrip(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, address string, domain string, wg *sync.WaitGroup, tk *TestKeys) {
	// TODO(bassosimone): make the timeout user-configurable
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	defer wg.Done()
	pings := []*SinglePing{}
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "DNSPing #%d %s %s", index, address, domain)
	// TODO(bassosimone, DecFox): what should we do if the user passes us a resolver with a
	// domain name in terms of saving its results? Shall we save also the system resolver's lookups?
	// Shall we, otherwise, pre-resolve the domain name to IP addresses once and for all? In such
	// a case, shall we use all the available IP addresses or just some of them?
	dialer := netxlite.NewDialerWithStdlibResolver(logger)
	resolver := trace.NewParallelUDPResolver(logger, dialer, address)
	_, err := resolver.LookupHost(ctx, domain)
	ol.Stop(err)
	for _, lookup := range trace.DNSLookupsFromRoundTrip() {
		if lookup.QueryType == "A" || lookup.QueryType == "AAAA" {
			pings = append(pings, &SinglePing{
				Query: lookup,
			})
		}
	}
	tk.addPings(pings)
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
