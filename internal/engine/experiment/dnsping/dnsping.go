// Package dnsping is the experimental dnsping experiment.
package dnsping

import (
	"context"
	"errors"
	"net/url"
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
	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`

	// Domain is the domain to test.
	Domain string `ooni:"domain is the domain to test"`
}

func (c Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

func (c Config) domain() string {
	if c.Domain != "" {
		return c.Domain
	}
	return "edge-chat.instagram.com"
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Pings []*SinglePing `json:"pings"`
}

// SinglePing contains the results of a single ping.
type SinglePing struct {
	DNSRoundTrips []*measurex.ArchivalDNSRoundTripEvent `json:"dns_round_trips"`
	Queries       []*measurex.ArchivalDNSLookupEvent    `json:"queries"`
	NetworkEvents []*measurex.ArchivalNetworkEvent      `json:"network_events"`
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
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	if measurement.Input == "" {
		return errors.New("no input provided")
	}
	parsed, err := url.Parse(string(measurement.Input))
	if err != nil {
		return errors.New("input is not an URL")
	}
	if parsed.Scheme != "udp" {
		return errors.New("we only support udp://<host>:<port> for now")
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	mx := measurex.NewMeasurerWithDefaultSettings()
	for i := int64(0); i < m.config.repetitions(); i++ {
		meas := mx.LookupHostUDP(ctx, m.config.domain(), parsed.Host)
		tk.Pings = append(tk.Pings, &SinglePing{
			DNSRoundTrips: measurex.NewArchivalDNSRoundTripEventList(meas.DNSRoundTrip),
			Queries:       measurex.NewArchivalDNSLookupEventList(meas.LookupHost),
			NetworkEvents: measurex.NewArchivalNetworkEventList(meas.ReadWrite),
		})
		<-ticker.C
	}
	return nil // return nil so we always submit the measurement
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
