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
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "dnsping"
	testVersion = "0.1.0"
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

// TestKeys contains the experiment results.
type TestKeys struct {
	Pings []*SinglePing `json:"pings"`
}

// TODO(bassosimone): save more data once the dnsping improvements at
// github.com/bassosimone/websteps-illustrated contains have been merged
// into this repository. When this happens, we'll able to save raw
// queries and network events of each individual query.

// SinglePing contains the results of a single ping.
type SinglePing struct {
	Queries []*measurex.ArchivalDNSLookupEvent `json:"queries"`
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
	tk := new(TestKeys)
	measurement.TestKeys = tk
	mxmx := measurex.NewMeasurerWithDefaultSettings()
	out := make(chan *measurex.DNSMeasurement)
	domains := strings.Split(m.config.domains(), " ")
	for _, domain := range domains {
		go m.dnsPingLoop(ctx, mxmx, parsed.Host, domain, out)
	}
	// The following multiplication could overflow but we're always using small
	// numbers so it's fine for us not to bother with checking for that
	numResults := int(m.config.repetitions()) * len(domains) * 2
	for len(tk.Pings) < numResults {
		meas := <-out
		// TODO(bassosimone): when we merge the improvements at
		// https://github.com/bassosimone/websteps-illustrated it
		// will become unnecessary to split with query type
		// as we're doing below.
		queries := measurex.NewArchivalDNSLookupEventList(meas.LookupHost)
		tk.Pings = append(tk.Pings, m.onlyQueryWithType(queries, "A")...)
		tk.Pings = append(tk.Pings, m.onlyQueryWithType(queries, "AAAA")...)
	}
	return nil // return nil so we always submit the measurement
}

// onlyQueryWithType returns only the queries with the given type.
func (m *Measurer) onlyQueryWithType(
	in []*measurex.ArchivalDNSLookupEvent, kind string) (out []*SinglePing) {
	for _, query := range in {
		if query.QueryType == kind {
			out = append(out, &SinglePing{
				Queries: []*measurex.ArchivalDNSLookupEvent{query},
			})
		}
	}
	return
}

// dnsPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) dnsPingLoop(ctx context.Context, mxmx *measurex.Measurer,
	address string, domain string, out chan<- *measurex.DNSMeasurement) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i := int64(0); i < m.config.repetitions(); i++ {
		go m.dnsPingAsync(ctx, mxmx, address, domain, out)
		<-ticker.C
	}
}

// dnsPingAsync performs a DNS ping and emits the result onto the out channel.
func (m *Measurer) dnsPingAsync(ctx context.Context, mxmx *measurex.Measurer,
	address string, domain string, out chan<- *measurex.DNSMeasurement) {
	out <- m.dnsRoundTrip(ctx, mxmx, address, domain)
}

// dnsRoundTrip performs a round trip and returns the results to the caller.
func (m *Measurer) dnsRoundTrip(ctx context.Context, mxmx *measurex.Measurer,
	address string, domain string) *measurex.DNSMeasurement {
	// TODO(bassosimone): make the timeout user-configurable
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return mxmx.LookupHostUDP(ctx, domain, address)
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
