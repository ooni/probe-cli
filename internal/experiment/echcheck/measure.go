package echcheck

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	testName    = "echcheck"
	testVersion = "0.2.0"
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
	addrs, err := resolver.LookupHost(ctx, parsed.Host)
	ol.Stop(err)
	if err != nil {
		return err
	}
	runtimex.Assert(len(addrs) > 0, "expected at least one entry in addrs")
	address := net.JoinHostPort(addrs[0], "443")

	handshakes := []func() (chan model.ArchivalTLSOrQUICHandshakeResult, error){
		// handshake with ECH disabled and SNI coming from the URL
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, trace, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, "", args.Session.Logger())
		},
		// handshake with ECH enabled and ClientHelloOuter SNI coming from the URL
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, trace, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, parsed.Host, args.Session.Logger())
		},
		// handshake with ECH enabled and hardcoded different ClientHelloOuter SNI
		func() (chan model.ArchivalTLSOrQUICHandshakeResult, error) {
			return connectAndHandshake(ctx, trace, args.Measurement.MeasurementStartTimeSaved,
				address, parsed.Host, "cloudflare.com", args.Session.Logger())
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
