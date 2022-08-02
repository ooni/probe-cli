// Package dnsscan is the experimental dnsscan experiment.
package dnsscan

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "dnsscan"
	testVersion = "0.1.0"
)

// TestKeys contains the experiment results.
type TestKeys struct {
	Queries        []*model.ArchivalDNSLookupResult `json:"queries"`
	TargetResolver string                           `json:"target_resolver"`
}

// Config contains the experiment configuration.
type Config struct {
	// Address of the DNS resolver to be used for testing
	Resolver string `ooni:"address of the DNS resolver to use"`
}

func (c Config) resolver() string {
	if c.Resolver != "" {
		return c.Resolver
	}
	return "udp://8.8.8.8:53"
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

	// errInputFormatInvalid means that the input format is not valid
	errInputFormatInvalid = errors.New("input is not a domain or URL")

	// errNoInputProvided indicates that the chosen DNS resolver is not currently supported
	errResolverNotSupported = errors.New("resolver not supported")

	// errInvalidResolver indicates that the chosen DNS resolver is invalid
	errInvalidResolver = errors.New("resolver address is invalid")

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

	// XXX do we also care to support URLs as inputs?
	domain := string(measurement.Input)

	tk := new(TestKeys)
	tk.TargetResolver = m.config.resolver()
	measurement.TestKeys = tk

	resolver := m.config.resolver()
	parsedResolver, err := url.Parse(string(resolver))
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidResolver, err.Error())
	}
	if parsedResolver.Scheme != "udp" {
		return errResolverNotSupported
	}
	if parsedResolver.Port() == "" {
		return errMissingPort
	}
	resolverIP := net.ParseIP(parsedResolver.Hostname())
	if resolverIP == nil {
		return fmt.Errorf("%w: invalid IP address", errInvalidResolver)
	}
	resolverAddress := fmt.Sprintf("%s:%s", resolverIP, parsedResolver.Port())
	// is an IPv6 address
	if resolverIP.To4() == nil {
		resolverAddress = fmt.Sprintf("[%s]:%s", resolverIP, parsedResolver.Port())
	}
	m.dnsRoundTrip(ctx, measurement.MeasurementStartTimeSaved, sess.Logger(), resolverAddress, domain, tk)
	return nil // return nil so we always submit the measurement
}

// dnsRoundTrip performs a round trip and returns the results to the caller.
func (m *Measurer) dnsRoundTrip(ctx context.Context, zeroTime time.Time,
	logger model.Logger, address string, domain string, tk *TestKeys) {
	fmt.Printf("Measuing %s %s", domain, address)
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	trace := measurexlite.NewTrace(0, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "DNSScan %s %s", address, domain)

	dialer := netxlite.NewDialerWithStdlibResolver(logger)
	resolver := trace.NewParallelUDPResolver(logger, dialer, address)
	_, err := resolver.LookupHost(ctx, domain)
	ol.Stop(err)
	// Add the dns.TypeA query
	tk.Queries = append(tk.Queries, <-trace.DNSLookup[dns.TypeA])
	// Add the dns.TypeAAAA query
	tk.Queries = append(tk.Queries, <-trace.DNSLookup[dns.TypeAAAA])
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
