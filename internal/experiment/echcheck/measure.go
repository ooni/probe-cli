package echcheck

import (
	"context"
	"errors"
	"net"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

	// 2. Set up TCP connections
	ol = logx.NewOperationLogger(args.Session.Logger(), "echcheck: TCPConnect#1 %s", address)
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	ol = logx.NewOperationLogger(args.Session.Logger(), "echcheck: TCPConnect#2 %s", address)
	conn2, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	ol = logx.NewOperationLogger(args.Session.Logger(), "echcheck: TCPConnect#3 %s", address)
	conn3, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	if err != nil {
		return netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ConnectOperation, err)
	}

	// 3. Conduct and measure control and target TLS handshakes in parallel
	noEchChannel := make(chan model.ArchivalTLSOrQUICHandshakeResult)
	echWithMatchingOuterSniChannel := make(chan model.ArchivalTLSOrQUICHandshakeResult)
	echWithExampleOuterSniChannel := make(chan model.ArchivalTLSOrQUICHandshakeResult)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	go func() {
		noEchChannel <- *handshake(
			ctx,
			conn,
			args.Measurement.MeasurementStartTimeSaved,
			address,
			parsed.Host,
			args.Session.Logger(),
		)
	}()

	go func() {
		echWithMatchingOuterSniChannel <- *handshakeWithEch(
			ctx,
			conn2,
			args.Measurement.MeasurementStartTimeSaved,
			address,
			parsed.Host,
			args.Session.Logger(),
		)
	}()

	exampleSni := "cloudflare.com"
	go func() {
		echWithExampleOuterSniChannel <- *handshakeWithEch(
			ctx,
			conn3,
			args.Measurement.MeasurementStartTimeSaved,
			address,
			exampleSni,
			args.Session.Logger(),
		)
	}()

	noEch := <-noEchChannel
	echWithMatchingOuterSni := <-echWithMatchingOuterSniChannel
	echWithMatchingOuterSni.ServerName = parsed.Host
	echWithMatchingOuterSni.OuterServerName = parsed.Host
	echWithExampleOuterSni := <-echWithExampleOuterSniChannel
	echWithExampleOuterSni.ServerName = parsed.Host
	echWithExampleOuterSni.OuterServerName = exampleSni

	args.Measurement.TestKeys = TestKeys{TLSHandshakes: []*model.ArchivalTLSOrQUICHandshakeResult{
		&noEch, &echWithMatchingOuterSni, &echWithExampleOuterSni,
	}}

	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}
