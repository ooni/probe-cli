// Package ntelegram contains the Telegram network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-020-telegram.md.
package ntelegram

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/dslx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName    = "ntelegram"
	testVersion = "0.1.0"
)

// Config contains the telegram experiment config.
type Config struct{}

// TestKeys contains telegram test keys.
type TestKeys struct {
	mu sync.Mutex

	Agent         string                   `json:"agent"`                // df-001-httpt
	SOCKSProxy    string                   `json:"socksproxy,omitempty"` // df-001-httpt
	Requests      []tracex.RequestEntry    `json:"requests"`             // df-001-httpt
	Queries       []tracex.DNSQueryEntry   `json:"queries"`              // df-002-dnst
	TCPConnect    []tracex.TCPConnectEntry `json:"tcp_connect"`          // df-005-tcpconnect
	TLSHandshakes []tracex.TLSHandshake    `json:"tls_handshakes"`       // df-006-tlshandshake
	NetworkEvents []tracex.NetworkEvent    `json:"network_events"`       // df-008-netevents

	TelegramHTTPBlocking bool    `json:"telegram_http_blocking"`
	TelegramTCPBlocking  bool    `json:"telegram_tcp_blocking"`
	TelegramWebFailure   *string `json:"telegram_web_failure"`
	TelegramWebStatus    string  `json:"telegram_web_status"`
}

// NewTestKeys creates new telegram TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		TelegramHTTPBlocking: false,
		TelegramTCPBlocking:  false,
		TelegramWebFailure:   nil,
		TelegramWebStatus:    "ok",
	}
}

// mergeObservations updates the TestKeys using the given [Observations] (goroutine safe).
func (tk *TestKeys) mergeObservations(obs []*dslx.Observations) {
	defer tk.mu.Unlock()
	tk.mu.Lock()
	for _, o := range obs {
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

// maybeSetDCFailure updates the TestKeys using the tcp and http success counters (goroutine safe).
func (tk *TestKeys) maybeSetDCFailure(
	tcpSuccessCounter *dslx.Counter[*dslx.TCPConnection],
	httpSuccessCounter *dslx.Counter[*dslx.HTTPResponse],
) {
	defer tk.mu.Unlock()
	tk.mu.Lock()
	tk.TelegramTCPBlocking = tcpSuccessCounter.Value() <= 0
	tk.TelegramHTTPBlocking = httpSuccessCounter.Value() <= 0
}

// setWebFailure updates the TestKeys using the given error (goroutine safe).
func (tk *TestKeys) setWebFailure(err error) {
	defer tk.mu.Unlock()
	tk.mu.Lock()
	tk.TelegramWebStatus = "blocked"
	tk.TelegramWebFailure = tracex.NewFailure(err)
}

// Measurer performs the measurement
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config
}

// ExperimentName implements ExperimentMeasurer.ExperimentName
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// measureDC measures telegram datacenter endpoints by issuing HTTP POST requests
// and calls wg.Done() upon return.
func measureDC(
	ctx context.Context,
	sess model.ExperimentSession,
	zeroTime time.Time,
	idGen *atomic.Int64,
	tk *TestKeys,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// ipAddrs contains the DCs IP addresses
	var ipAddrs = dslx.NewAddressSet().Add(
		"149.154.175.50",
		"149.154.167.51",
		"149.154.175.100",
		"149.154.167.91",
		"149.154.171.5",
		"95.161.76.100",
	)
	// construct the list of endpoints to measure: we need to
	// measure each IP address with port 80 and 443
	var (
		endpoints []*dslx.Endpoint
		ports     = []int{80, 443}
	)
	for addr := range ipAddrs.M {
		for _, port := range ports {
			endpoints = append(endpoints, dslx.NewEndpoint(
				dslx.EndpointNetwork("tcp"),
				dslx.EndpointAddress(net.JoinHostPort(addr, strconv.Itoa(port))),
				dslx.EndpointOptionIDGenerator(idGen),
				dslx.EndpointOptionLogger(sess.Logger()),
				dslx.EndpointOptionZeroTime(zeroTime),
			))
		}
	}

	connpool := &dslx.ConnPool{}
	defer connpool.Close()
	var (
		tcpSuccessCounter  = dslx.Counter[*dslx.TCPConnection]{}
		httpSuccessCounter = dslx.Counter[*dslx.HTTPResponse]{}
	)

	// construct the http/POST function to measure the endpoints
	httpFunc := dslx.Compose5(
		dslx.TCPConnect(connpool),
		tcpSuccessCounter.Func(), // count number of successful TCP connects
		dslx.HTTPTransportTCP(),
		dslx.HTTPRequest(
			dslx.HTTPRequestOptionMethod("POST"),
		),
		httpSuccessCounter.Func(), // count number of successful HTTP roundtrips
	)
	// measure all the endpoints in parallel and collect the results
	results := dslx.Map(
		ctx,
		dslx.Parallelism(3),
		httpFunc,
		dslx.StreamList(endpoints...),
	)
	coll := dslx.Collect(results)
	tk.mergeObservations(dslx.ExtractObservations(coll...))
	tk.maybeSetDCFailure(&tcpSuccessCounter, &httpSuccessCounter)
}

// measureWeb measures Telegram Web and calls wg.Done() upon return.
func measureWeb(
	ctx context.Context,
	sess model.ExperimentSession,
	zeroTime time.Time,
	idGen *atomic.Int64,
	tk *TestKeys,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// describe the DNS measurement input
	webDomain := "web.telegram.org"
	dnsInput := dslx.NewDomainToResolve(
		dslx.DomainName(webDomain),
		dslx.DNSLookupOptionIDGenerator(idGen),
		dslx.DNSLookupOptionLogger(sess.Logger()),
		dslx.DNSLookupOptionZeroTime(zeroTime),
	)
	// construct getaddrinfo resolver
	lookup := dslx.DNSLookupGetaddrinfo()

	// run the DNS Lookup
	dnsResult := lookup.Apply(ctx, dnsInput)

	// extract and merge observations with the test keys
	tk.mergeObservations(dslx.ExtractObservations(dnsResult))

	// if the lookup failed we return
	if dnsResult.Error != nil {
		tk.setWebFailure(dnsResult.Error)
		return
	}

	// obtain a unique set of IP addresses w/o bogons inside it
	ipAddrs := dslx.NewAddressSet(dnsResult).RemoveBogons()

	// create the set of endpoints
	endpoints := ipAddrs.ToEndpoints(
		dslx.EndpointNetwork("tcp"),
		dslx.EndpointPort(443),
		dslx.EndpointOptionDomain(webDomain),
		dslx.EndpointOptionIDGenerator(idGen),
		dslx.EndpointOptionLogger(sess.Logger()),
		dslx.EndpointOptionZeroTime(zeroTime),
	)

	connpool := &dslx.ConnPool{}
	defer connpool.Close()
	successes := dslx.Counter[*dslx.HTTPResponse]{}

	// create function for the https measurement
	httpsFunction := dslx.Compose5(
		dslx.TCPConnect(connpool),
		dslx.TLSHandshake(connpool),
		dslx.HTTPTransportTLS(),
		dslx.HTTPRequest(),
		successes.Func(), // count the number of successes
	)

	// run https measurement and collect the results
	httpsResults := dslx.Map(
		ctx,
		dslx.Parallelism(2),
		httpsFunction,
		dslx.StreamList(endpoints...),
	)
	coll := dslx.Collect(httpsResults)
	tk.mergeObservations(dslx.ExtractObservations(coll...))

	firstError, _ := dslx.FirstError(coll...)
	if firstError != nil {
		tk.setWebFailure(firstError)
	}
}

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	sess := args.Session
	idGen := &atomic.Int64{}
	zeroTime := time.Now()

	tk := NewTestKeys()
	tk.Agent = "redirect"
	measurement := args.Measurement
	measurement.TestKeys = tk

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go measureDC(ctx, sess, zeroTime, idGen, tk, wg)

	wg.Add(1)
	go measureWeb(ctx, sess, zeroTime, idGen, tk, wg)

	wg.Wait()
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	HTTPBlocking bool `json:"telegram_http_blocking"`
	TCPBlocking  bool `json:"telegram_tcp_blocking"`
	WebBlocking  bool `json:"telegram_web_blocking"`
	IsAnomaly    bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	tcpBlocking := tk.TelegramTCPBlocking
	httpBlocking := tk.TelegramHTTPBlocking
	webBlocking := tk.TelegramWebFailure != nil
	sk.TCPBlocking = tcpBlocking
	sk.HTTPBlocking = httpBlocking
	sk.WebBlocking = webBlocking
	sk.IsAnomaly = webBlocking || httpBlocking || tcpBlocking
	return sk, nil
}
