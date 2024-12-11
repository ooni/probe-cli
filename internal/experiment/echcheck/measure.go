package echcheck

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	testName    = "echcheck"
	testVersion = "0.3.0"
	defaultURL  = "https://cloudflare-ech.com/cdn-cgi/trace"
)

var (
	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidInputScheme indicates that the input scheme is invalid
	errInvalidInputScheme = errors.New("input scheme must be https")
)

// TestKeys contains echcheck test keys.
type TestKeys struct {
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
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
		args.Measurement.Input = defaultURL
	}
	parsed, err := url.Parse(string(args.Measurement.Input))
	if err != nil {
		return errInputIsNotAnURL
	}
	if parsed.Scheme != "https" {
		return errInvalidInputScheme
	}

	// 1. perform a DNSLookup
	ol := logx.NewOperationLogger(args.Session.Logger(), "echcheck: DNSLookup[%s] %s", m.config.resolverURL(), parsed.Host)
	trace := measurexlite.NewTrace(0, args.Measurement.MeasurementStartTimeSaved)
	resolver := trace.NewParallelDNSOverHTTPSResolver(args.Session.Logger(), m.config.resolverURL())
	// We dial the alias, even when there are hints in the HTTPS record.
	addrs, addrsErr := resolver.LookupHost(ctx, parsed.Host)
	httpsRr, httpsErr := resolver.LookupHTTPS(ctx, parsed.Host)
	ol.Stop(err)
	if addrsErr != nil {
		return addrsErr
	}
	if httpsErr != nil {
		return httpsErr
	}
	rawEchConfig := httpsRr.Ech
	ecl, err := parseRawEchConfig(rawEchConfig)
	if err != nil {
		return fmt.Errorf("failed to parse ECH config: %w", err)
	}
	if len(ecl.Configs) == 0 {
		return fmt.Errorf("no ECH configs for %s", parsed.Host)
	}
	outerServerName := string(ecl.Configs[0].PublicName)
	for _, ec := range ecl.Configs {
		if string(ec.PublicName) != outerServerName {
			// It's perfectly valid to have multiple ECH configs with different
			// `PublicName`s. But, since we can't see which one is selected by
			// go's tls package, we can't accurately record OuterServerName.
			return fmt.Errorf("ambigious OuterServerName for %s", parsed.Host)
		}
	}

	runtimex.Assert(len(addrs) > 0, "expected at least one entry in addrs")
	address := net.JoinHostPort(addrs[0], "443")

	handshakes := []func() (chan model.ArchivalTLSOrQUICHandshakeResult, error){
		// Handshake with no ECH
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, NoECH, echConfigList{}, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, args.Session.Logger())
		},

		// Handshake with ECH GREASE
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, GreaseECH, echConfigList{}, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, args.Session.Logger())
		},

		// Use real ECH
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, RealECH, ecl, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, args.Session.Logger())
		},
	}

	// We shuffle the order in which the operations are done to avoid residual
	// censorship issues.
	rand.Shuffle(len(handshakes), func(i, j int) {
		handshakes[i], handshakes[j] = handshakes[j], handshakes[i]
	})

	var channels [3](chan model.ArchivalTLSOrQUICHandshakeResult)
	var results [3](model.ArchivalTLSOrQUICHandshakeResult)

	// Fire the handshakes in parallel
	// TODO: currently if one of the connects fails we fail the whole result
	// set. This is probably OK given that we only ever use the same address,
	// but this may be something we want to change in the future.
	for idx, hs := range handshakes {
		channels[idx], err = hs()
		if err != nil {
			return err
		}
	}

	// Wait on each channel for the results to come in
	for idx, ch := range channels {
		results[idx] = <-ch
	}

	args.Measurement.TestKeys = TestKeys{TLSHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{
		&results[0], &results[1], &results[2],
	}}

	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}
