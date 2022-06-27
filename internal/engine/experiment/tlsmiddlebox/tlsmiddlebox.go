package tlsmiddlebox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlsmiddlebox/internal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	testName    = "tlsmiddlebox"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	// ResolverURL is the default DoH resolver
	ResolverURL string `ooni:"URL for DoH resolver"`

	// SNIPass is the SNI value we don't expect to be blocked
	SNIPass string `ooni:"the SNI value to cal"`

	// Delay is the delay between each iteration (in milliseconds).
	Delay int64 `ooni:"delay between consecutive iterations"`

	// Iterations is the default number of interations we trace
	Iterations int `ooni:"iterations is the number of iterations"`

	// SNI is the SNI value to use.
	SNI string `ooni:"the SNI value to use"`
}

func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}

func (c Config) snipass() string {
	if c.SNIPass != "" {
		return c.SNIPass
	}
	return "google.com"
}

func (c Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return 100 * time.Millisecond
}

func (c Config) iterations() int {
	if c.Iterations > 0 {
		return c.Iterations
	}
	return 20
}

func (c Config) sni(address string) string {
	if c.SNI != "" {
		return c.SNI
	}
	return address
}

// TestKeys contains the experiment results.
type TestKeys struct {
	DNSLookUp  *model.ArchivalDNSLookupResult    `json:"dns_lookup"`
	TCPConnect []*model.ArchivalTCPConnectResult `json:"tcp_connect"`
	TLSTrace   []*CompleteTrace                  `json:"tls_trace"`
}

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
	tk := new(TestKeys)
	measurement.TestKeys = tk
	sni := m.config.sni(parsed.Host)
	// 1. perform a DNSLookUp
	outDNS, addrs, err := m.DNSLookup(ctx, parsed.Hostname(), nil)
	tk.DNSLookUp = outDNS
	if err != nil {
		return err
	}
	// 2. measure addresses
	m.MeasureAddrs(ctx, addrs, parsed.Port(), sni, tk)
	return nil
}

// MeasureAddrs measures the array of addresses obtained from DNSLookUp
func (m *Measurer) MeasureAddrs(ctx context.Context, addrs []string, port string,
	sni string, tk *TestKeys) {
	tcpEvents := make(chan *model.ArchivalTCPConnectResult, len(addrs))
	tlsEvents := make(chan *CompleteTrace, 2*len(addrs))
	wg := new(sync.WaitGroup)
	addrs = internal.PrepareAddrs(addrs, port)
	for _, addr := range addrs {
		wg.Add(1)
		go m.MeasureSingleAddr(ctx, addr, sni, tcpEvents, tlsEvents, wg)
	}
	wg.Wait()
	tk.TCPConnect = GetTCPEvents(tcpEvents)
	tk.TLSTrace = GetTLSEvents(tlsEvents)
}

// MeasureSingleAddr measures a single address
func (m *Measurer) MeasureSingleAddr(ctx context.Context, addr string,
	sni string, tcpEvents chan<- *model.ArchivalTCPConnectResult,
	tlsEvents chan<- *CompleteTrace, wg *sync.WaitGroup) error {
	defer wg.Done()
	err := m.MeasureTCP(ctx, addr, tcpEvents)
	if err != nil {
		return err
	}
	m.MeasureTLS(ctx, addr, sni, tlsEvents)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) *Measurer {
	return &Measurer{config: config}
}

type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
