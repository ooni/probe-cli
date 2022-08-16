package tlsmiddlebox

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
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
	SNIControl string `ooni:"the SNI value to cal"`

	// Delay is the delay between each iteration (in milliseconds).
	Delay int64 `ooni:"delay between consecutive iterations"`

	// Iterations is the default number of interations we trace
	MaxTTL int64 `ooni:"iterations is the number of iterations"`

	// SNI is the SNI value to use.
	SNI string `ooni:"the SNI value to use"`

	// ClientId is the client fingerprint to use
	ClientId int `ooni:"the ClientHello fingerprint to use"`
}

func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}

func (c Config) snicontrol() string {
	if c.SNIControl != "" {
		return c.SNIControl
	}
	return "example.com"
}

func (c Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return 100 * time.Millisecond
}

func (c Config) maxttl() int64 {
	if c.MaxTTL > 0 {
		return c.MaxTTL
	}
	return 20
}

func (c Config) sni(address string) string {
	if c.SNI != "" {
		return c.SNI
	}
	return address
}

func (c Config) clientid() int {
	if c.ClientId > 0 {
		return c.ClientId
	}
	return 0
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

// DNSLookup performs a DNS Lookup for the passed domain
func (m *Measurer) DNSLookup(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, domain string, tk *TestKeys) ([]string, error) {
	url := m.config.resolverURL()
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "DNSLookup #%d, %s, %s", index, url, domain)
	// TODO(DecFox): We are currently using the DoH resolver, we will
	// switch to the TRR2 resolver once we have it in measurexlite
	// Issue: https://github.com/ooni/probe/issues/2185
	resolver := trace.NewParallelDNSOverHTTPSResolver(logger, url)
	addrs, err := resolver.LookupHost(ctx, domain)
	ol.Stop(err)
	tk.addQueries(trace.DNSLookupsFromRoundTrip())
	return addrs, err
}

// TraceAddress measures a single address
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

// TCPConnect performs a TCP connect to filter working addresses
func (m *Measurer) TCPConnect(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, address string, tk *TestKeys) error {
	trace := measurexlite.NewTrace(index, zeroTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	ol := measurexlite.NewOperationLogger(logger, "TCPConnect #%d %s", index, address)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	tcpEvents := trace.TCPConnects()
	tk.addTCPConnect(tcpEvents)
	return err
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) *Measurer {
	return &Measurer{config: config}
}

type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
// TODO(DecFox): Add anamoly logic to generate summary keys for the experiment
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
