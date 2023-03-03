// Package nsniblocking contains the SNI blocking network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-024-sni-blocking.md.
package nsniblocking

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/dslx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName    = "sni_blocking"
	testVersion = "0.3.0"
)

// Config contains the experiment config.
type Config struct {
	// ControlSNI is the SNI to be used for the control.
	ControlSNI string

	// TestHelperAddress is the address of the test helper.
	TestHelperAddress string

	// ResolverURL is the URL describing the resolver to use.
	ResolverURL string
}

// Subresult contains the keys of a single measurement
// that targets either the target or the control.
type Subresult struct {
	Failure       *string                  `json:"failure"`
	NetworkEvents []tracex.NetworkEvent    `json:"network_events"`
	Queries       []tracex.DNSQueryEntry   `json:"queries"`
	Requests      []tracex.RequestEntry    `json:"requests"`
	TCPConnect    []tracex.TCPConnectEntry `json:"tcp_connect"`
	TLSHandshakes []tracex.TLSHandshake    `json:"tls_handshakes"`
	Cached        bool                     `json:"-"`
	SNI           string                   `json:"sni"`
	THAddress     string                   `json:"th_address"`
}

func (tk *Subresult) MergeObservations(obs []*dslx.Observations) {
	for _, o := range obs {
		// update the easy to update entries first
		for _, e := range o.NetworkEvents {
			tk.NetworkEvents = append(tk.NetworkEvents, *e)
		}
		for _, e := range o.Queries {
			tk.Queries = append(tk.Queries, *e)
		}
		for _, e := range o.Requests {
			tk.Requests = append(tk.Requests, *e)
		}
		for _, e := range o.TCPConnect {
			tk.TCPConnect = append(tk.TCPConnect, *e)
		}
		for _, e := range o.TLSHandshakes {
			tk.TLSHandshakes = append(tk.TLSHandshakes, *e)
		}
	}
}

// TestKeys contains sniblocking test keys.
type TestKeys struct {
	Control Subresult `json:"control"`
	Result  string    `json:"result"`
	Target  Subresult `json:"target"`
}

const (
	classAnomalyTestHelperUnreachable   = "anomaly.test_helper_unreachable"
	classAnomalyTimeout                 = "anomaly.timeout"
	classAnomalyUnexpectedFailure       = "anomaly.unexpected_failure"
	classInterferenceClosed             = "interference.closed"
	classInterferenceInvalidCertificate = "interference.invalid_certificate"
	classInterferenceReset              = "interference.reset"
	classInterferenceUnknownAuthority   = "interference.unknown_authority"
	classSuccessGotServerHello          = "success.got_server_hello"
)

func (tk *TestKeys) classify() string {
	if tk.Target.Failure == nil {
		return classSuccessGotServerHello
	}
	switch *tk.Target.Failure {
	case netxlite.FailureConnectionRefused:
		return classAnomalyTestHelperUnreachable
	case netxlite.FailureConnectionReset:
		return classInterferenceReset
	case netxlite.FailureDNSNXDOMAINError, netxlite.FailureAndroidDNSCacheNoData:
		return classAnomalyTestHelperUnreachable
	case netxlite.FailureEOFError:
		return classInterferenceClosed
	case netxlite.FailureGenericTimeoutError:
		if tk.Control.Failure != nil {
			return classAnomalyTestHelperUnreachable
		}
		return classAnomalyTimeout
	case netxlite.FailureSSLInvalidCertificate:
		return classInterferenceInvalidCertificate
	case netxlite.FailureSSLInvalidHostname:
		return classSuccessGotServerHello
	case netxlite.FailureSSLUnknownAuthority:
		return classInterferenceUnknownAuthority
	}
	return classAnomalyUnexpectedFailure
}

// Measurer performs the measurement.
type Measurer struct {
	cache  map[string]Subresult
	config Config
	mu     sync.Mutex
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) measureone(
	ctx context.Context,
	sess model.ExperimentSession,
	beginning time.Time,
	sni string,
	thaddr string,
) Subresult {
	// slightly delay the measurement
	gen := rand.New(rand.NewSource(time.Now().UnixNano()))
	sleeptime := time.Duration(gen.Intn(250)) * time.Millisecond
	select {
	case <-time.After(sleeptime):
	case <-ctx.Done():
		s := netxlite.FailureInterrupted
		return Subresult{
			Failure:   &s,
			THAddress: thaddr,
			SNI:       sni,
		}
	}

	idGen := &atomic.Int64{}
	zeroTime := time.Now()

	// describe the DNS measurement input
	thaddrHost, _, _ := net.SplitHostPort(thaddr) // TODO: handle error?
	dnsInput := dslx.NewDomainToResolve(
		dslx.DomainName(thaddrHost),
		dslx.DNSLookupOptionIDGenerator(idGen),
		dslx.DNSLookupOptionLogger(sess.Logger()),
		dslx.DNSLookupOptionZeroTime(zeroTime),
	)
	// construct resolver
	lookup := dslx.DNSLookupGetaddrinfo()
	if m.config.ResolverURL != "" {
		lookup = dslx.DNSLookupUDP(m.config.ResolverURL)
	}

	// run the DNS Lookup
	dnsResult := lookup.Apply(ctx, dnsInput)

	// create a subresult, extract and merge observations
	// Create the subresult
	subresult := Subresult{
		SNI:       sni,
		THAddress: thaddr,
	}
	subresult.MergeObservations(dslx.ExtractObservations(dnsResult))

	// if the lookup has failed we return
	if dnsResult.Error != nil {
		return subresult
	}

	// obtain a unique set of IP addresses w/o bogons inside it
	ipAddrs := dslx.NewAddressSet(dnsResult).RemoveBogons()

	// create the set of endpoints
	endpoints := ipAddrs.ToEndpoints(
		dslx.EndpointNetwork("tcp"),
		dslx.EndpointPort(443),
		dslx.EndpointOptionDomain(thaddr),
		dslx.EndpointOptionIDGenerator(idGen),
		dslx.EndpointOptionLogger(sess.Logger()),
		dslx.EndpointOptionZeroTime(zeroTime),
	)

	// create the established connections pool
	connpool := &dslx.ConnPool{}
	defer connpool.Close()

	// count the number of successes
	successes := dslx.Counter[*dslx.TLSConnection]{}

	// run tls handshake measurement
	httpsResults := dslx.Map(
		ctx,
		dslx.Parallelism(2),
		dslx.Compose3(
			dslx.TCPConnect(connpool),
			dslx.TLSHandshake(
				connpool,
				dslx.TLSHandshakeOptionServerName(sni),
			),
			successes.Func(), // number of times we arrive here
		),
		dslx.StreamList(endpoints...),
	)

	coll := dslx.Collect(httpsResults)

	// extract and merge observations
	subresult.MergeObservations(dslx.ExtractObservations(coll...))

	// extract first error
	firstError, _ := dslx.FirstError(coll...)
	if firstError != nil {
		subresult.Failure = tracex.NewFailure(firstError)
	}

	return subresult
}

func (m *Measurer) measureonewithcache(
	ctx context.Context,
	output chan<- Subresult,
	sess model.ExperimentSession,
	beginning time.Time,
	sni string,
	thaddr string,
) {
	cachekey := sni + thaddr
	m.mu.Lock()
	smk, okay := m.cache[cachekey]
	m.mu.Unlock()
	if okay {
		output <- smk
		return
	}
	smk = m.measureone(ctx, sess, beginning, sni, thaddr)
	output <- smk
	smk.Cached = true
	m.mu.Lock()
	m.cache[cachekey] = smk
	m.mu.Unlock()
}

func (m *Measurer) startall(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, inputs []string,
) <-chan Subresult {
	outputs := make(chan Subresult, len(inputs))
	for _, input := range inputs {
		go m.measureonewithcache(
			ctx, outputs, sess,
			measurement.MeasurementStartTimeSaved,
			input, m.config.TestHelperAddress,
		)
	}
	return outputs
}

func processall(
	outputs <-chan Subresult,
	measurement *model.Measurement,
	inputs []string,
	sess model.ExperimentSession,
	controlSNI string,
) *TestKeys {
	var (
		current  int
		testkeys = new(TestKeys)
	)
	for smk := range outputs {
		if smk.SNI == controlSNI {
			testkeys.Control = smk
		} else if smk.SNI == string(measurement.Input) {
			testkeys.Target = smk
		} else {
			panic("unexpected smk.SNI")
		}
		current++
		sess.Logger().Debugf(
			"sni_blocking: %s: %s [cached: %+v]", smk.SNI,
			asString(smk.Failure), smk.Cached)
		if current >= len(inputs) {
			break
		}
	}
	testkeys.Result = testkeys.classify()
	sess.Logger().Infof("sni_blocking: result: %s", testkeys.Result)
	return testkeys
}

// maybeURLToSNI handles the case where the input is from the test-lists
// and hence every input is a URL rather than a domain.
func maybeURLToSNI(input model.MeasurementTarget) (model.MeasurementTarget, error) {
	parsed, err := url.Parse(string(input))
	if err != nil {
		return "", err
	}
	if parsed.Path == string(input) {
		return input, nil
	}
	return model.MeasurementTarget(parsed.Hostname()), nil
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	measurement := args.Measurement
	sess := args.Session
	m.mu.Lock()
	if m.cache == nil {
		m.cache = make(map[string]Subresult)
	}
	m.mu.Unlock()
	if m.config.ControlSNI == "" {
		m.config.ControlSNI = "example.org"
	}
	if measurement.Input == "" {
		return errors.New("Experiment requires measurement.Input")
	}
	if m.config.TestHelperAddress == "" {
		m.config.TestHelperAddress = net.JoinHostPort(
			m.config.ControlSNI, "443",
		)
	}
	// TODO: urlgetter.RegisterExtensions. Do we need to replace that?

	// TODO(bassosimone): if the user has configured DoT or DoH, here we
	// probably want to perform the name resolution before the measurements
	// or to make sure that the classify logic is robust to that.
	//
	// See https://github.com/ooni/probe-engine/issues/392.
	maybeParsed, err := maybeURLToSNI(measurement.Input)
	if err != nil {
		return err
	}
	measurement.Input = maybeParsed
	inputs := []string{m.config.ControlSNI}
	if string(measurement.Input) != m.config.ControlSNI {
		inputs = append(inputs, string(measurement.Input))
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second*time.Duration(len(inputs)))
	defer cancel()
	outputs := m.startall(ctx, sess, measurement, inputs)
	measurement.TestKeys = processall(
		outputs, measurement, inputs, sess, m.config.ControlSNI,
	)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

func asString(failure *string) (result string) {
	result = "success"
	if failure != nil {
		result = *failure
	}
	return
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
