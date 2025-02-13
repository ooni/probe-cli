package echcheck

import (
	"context"
	crand "crypto/rand"
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
	NetworkEvents []*model.ArchivalNetworkEvent             `json:"network_events"`
	Queries       []*model.ArchivalDNSLookupResult          `json:"queries"`
	TCPConnects   []*model.ArchivalTCPConnectResult         `json:"tcp_connects"`
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

	// DNS Lookups for Address and HTTPS RR
	ol := logx.NewOperationLogger(args.Session.Logger(), "echcheck: DNSLookups[%s] %s", m.config.resolverURL(), parsed.Host)
	trace := measurexlite.NewTrace(0, args.Measurement.MeasurementStartTimeSaved)
	resolver := trace.NewParallelDNSOverHTTPSResolver(args.Session.Logger(), m.config.resolverURL())
	// We dial the alias, even when there are hints in the HTTPS record.
	addrs, addrsErr := resolver.LookupHost(ctx, parsed.Hostname())
	// Port prefixing per:
	// https://www.rfc-editor.org/rfc/rfc9460.html#name-query-names-for-https-rrs
	var dnsQueryHost = parsed.Hostname()
	if parsed.Port() != "" && parsed.Port() != "443" {
		dnsQueryHost = fmt.Sprintf("_%s._https.%s", parsed.Port(), parsed.Hostname())
	}
	httpsRr, httpsErr := resolver.LookupHTTPS(ctx, dnsQueryHost)
	ol.Stop(err)

	if addrsErr != nil {
		return addrsErr
	}
	if httpsErr != nil {
		return httpsErr
	}
	realEchConfig := httpsRr.Ech
	grease, err := generateGreaseyECHConfigList(crand.Reader, parsed.Hostname())
	if err != nil {
		return fmt.Errorf("failed to generate GREASE ECH config: %w", err)
	}

	runtimex.Assert(len(addrs) > 0, "expected at least one entry in addrs")
	port := parsed.Port()
	if port == "" {
		port = "443"
	}
	address := net.JoinHostPort(addrs[0], port)

	handshakes := []func() (chan TestKeys, error){
		// Handshake with no ECH
		func() (chan TestKeys, error) {
			return startHandshake(ctx, []byte{}, false,
				args.Measurement.MeasurementStartTimeSaved, address,
				parsed, args.Session.Logger(), nil)
		},

		// Handshake with ECH GREASE
		func() (chan TestKeys, error) {
			return startHandshake(ctx, grease, true,
				args.Measurement.MeasurementStartTimeSaved, address,
				parsed, args.Session.Logger(), nil)
		},

		// Handshake with real ECH
		func() (chan TestKeys, error) {
			return startHandshake(ctx, realEchConfig, false,
				args.Measurement.MeasurementStartTimeSaved, address,
				parsed, args.Session.Logger(), nil)
		},
	}

	// We shuffle the order in which the operations are done to avoid residual
	// censorship issues.
	rand.Shuffle(len(handshakes), func(i, j int) {
		handshakes[i], handshakes[j] = handshakes[j], handshakes[i]
	})

	var channels [3](chan TestKeys)

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

	alltks := TestKeys{
		TLSHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{},
		NetworkEvents: trace.NetworkEvents(),
		Queries:       trace.DNSLookupsFromRoundTrip(),
		TCPConnects:   []*model.ArchivalTCPConnectResult{},
	}

	// Wait on each channel for the results to come in
	for _, ch := range channels {
		tk := <-ch
		alltks.TLSHandshakes = append(alltks.TLSHandshakes, tk.TLSHandshakes...)
		alltks.NetworkEvents = append(alltks.NetworkEvents, tk.NetworkEvents...)
		alltks.Queries = append(alltks.Queries, tk.Queries...)
		alltks.TCPConnects = append(alltks.TCPConnects, tk.TCPConnects...)
	}

	args.Measurement.TestKeys = alltks

	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}
