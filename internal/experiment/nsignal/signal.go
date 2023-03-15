// Package nsignal contains the Signal network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-029-signal.md.
package nsignal

import (
	"context"
	"crypto/x509"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/dslx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName    = "nsignal"
	testVersion = "0.1.0"

	signalCA = `-----BEGIN CERTIFICATE-----
MIID7zCCAtegAwIBAgIJAIm6LatK5PNiMA0GCSqGSIb3DQEBBQUAMIGNMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKQ2FsaWZvcm5pYTEWMBQGA1UEBwwNU2FuIEZyYW5j
aXNjbzEdMBsGA1UECgwUT3BlbiBXaGlzcGVyIFN5c3RlbXMxHTAbBgNVBAsMFE9w
ZW4gV2hpc3BlciBTeXN0ZW1zMRMwEQYDVQQDDApUZXh0U2VjdXJlMB4XDTEzMDMy
NTIyMTgzNVoXDTIzMDMyMzIyMTgzNVowgY0xCzAJBgNVBAYTAlVTMRMwEQYDVQQI
DApDYWxpZm9ybmlhMRYwFAYDVQQHDA1TYW4gRnJhbmNpc2NvMR0wGwYDVQQKDBRP
cGVuIFdoaXNwZXIgU3lzdGVtczEdMBsGA1UECwwUT3BlbiBXaGlzcGVyIFN5c3Rl
bXMxEzARBgNVBAMMClRleHRTZWN1cmUwggEiMA0GCSqGSIb3DQEBAQUAA4IBDwAw
ggEKAoIBAQDBSWBpOCBDF0i4q2d4jAXkSXUGpbeWugVPQCjaL6qD9QDOxeW1afvf
Po863i6Crq1KDxHpB36EwzVcjwLkFTIMeo7t9s1FQolAt3mErV2U0vie6Ves+yj6
grSfxwIDAcdsKmI0a1SQCZlr3Q1tcHAkAKFRxYNawADyps5B+Zmqcgf653TXS5/0
IPPQLocLn8GWLwOYNnYfBvILKDMItmZTtEbucdigxEA9mfIvvHADEbteLtVgwBm9
R5vVvtwrD6CCxI3pgH7EH7kMP0Od93wLisvn1yhHY7FuYlrkYqdkMvWUrKoASVw4
jb69vaeJCUdU+HCoXOSP1PQcL6WenNCHAgMBAAGjUDBOMB0GA1UdDgQWBBQBixjx
P/s5GURuhYa+lGUypzI8kDAfBgNVHSMEGDAWgBQBixjxP/s5GURuhYa+lGUypzI8
kDAMBgNVHRMEBTADAQH/MA0GCSqGSIb3DQEBBQUAA4IBAQB+Hr4hC56m0LvJAu1R
K6NuPDbTMEN7/jMojFHxH4P3XPFfupjR+bkDq0pPOU6JjIxnrD1XD/EVmTTaTVY5
iOheyv7UzJOefb2pLOc9qsuvI4fnaESh9bhzln+LXxtCrRPGhkxA1IMIo3J/s2WF
/KVYZyciu6b4ubJ91XPAuBNZwImug7/srWvbpk0hq6A6z140WTVSKtJG7EP41kJe
/oF4usY5J7LPkxK3LWzMJnb5EIJDmRvyH8pyRwWg6Qm6qiGFaI4nL8QU4La1x2en
4DGXRaLMPRwjELNgQPodR38zoCMuA8gHZfZYYoZ7D7Q1wNUiVHcxuFrEeBaYJbLE
rwLV
-----END CERTIFICATE-----`

	signalCANew = `
-----BEGIN CERTIFICATE-----
MIIF2zCCA8OgAwIBAgIUAMHz4g60cIDBpPr1gyZ/JDaaPpcwDQYJKoZIhvcNAQEL
BQAwdTELMAkGA1UEBhMCVVMxEzARBgNVBAgTCkNhbGlmb3JuaWExFjAUBgNVBAcT
DU1vdW50YWluIFZpZXcxHjAcBgNVBAoTFVNpZ25hbCBNZXNzZW5nZXIsIExMQzEZ
MBcGA1UEAxMQU2lnbmFsIE1lc3NlbmdlcjAeFw0yMjAxMjYwMDQ1NTFaFw0zMjAx
MjQwMDQ1NTBaMHUxCzAJBgNVBAYTAlVTMRMwEQYDVQQIEwpDYWxpZm9ybmlhMRYw
FAYDVQQHEw1Nb3VudGFpbiBWaWV3MR4wHAYDVQQKExVTaWduYWwgTWVzc2VuZ2Vy
LCBMTEMxGTAXBgNVBAMTEFNpZ25hbCBNZXNzZW5nZXIwggIiMA0GCSqGSIb3DQEB
AQUAA4ICDwAwggIKAoICAQDEecifxMHHlDhxbERVdErOhGsLO08PUdNkATjZ1kT5
1uPf5JPiRbus9F4J/GgBQ4ANSAjIDZuFY0WOvG/i0qvxthpW70ocp8IjkiWTNiA8
1zQNQdCiWbGDU4B1sLi2o4JgJMweSkQFiyDynqWgHpw+KmvytCzRWnvrrptIfE4G
PxNOsAtXFbVH++8JO42IaKRVlbfpe/lUHbjiYmIpQroZPGPY4Oql8KM3o39ObPnT
o1WoM4moyOOZpU3lV1awftvWBx1sbTBL02sQWfHRxgNVF+Pj0fdDMMFdFJobArrL
VfK2Ua+dYN4pV5XIxzVarSRW73CXqQ+2qloPW/ynpa3gRtYeGWV4jl7eD0PmeHpK
OY78idP4H1jfAv0TAVeKpuB5ZFZ2szcySxrQa8d7FIf0kNJe9gIRjbQ+XrvnN+ZZ
vj6d+8uBJq8LfQaFhlVfI0/aIdggScapR7w8oLpvdflUWqcTLeXVNLVrg15cEDwd
lV8PVscT/KT0bfNzKI80qBq8LyRmauAqP0CDjayYGb2UAabnhefgmRY6aBE5mXxd
byAEzzCS3vDxjeTD8v8nbDq+SD6lJi0i7jgwEfNDhe9XK50baK15Udc8Cr/ZlhGM
jNmWqBd0jIpaZm1rzWA0k4VwXtDwpBXSz8oBFshiXs3FD6jHY2IhOR3ppbyd4qRU
pwIDAQABo2MwYTAOBgNVHQ8BAf8EBAMCAQYwDwYDVR0TAQH/BAUwAwEB/zAdBgNV
HQ4EFgQUtfNLxuXWS9DlgGuMUMNnW7yx83EwHwYDVR0jBBgwFoAUtfNLxuXWS9Dl
gGuMUMNnW7yx83EwDQYJKoZIhvcNAQELBQADggIBABUeiryS0qjykBN75aoHO9bV
PrrX+DSJIB9V2YzkFVyh/io65QJMG8naWVGOSpVRwUwhZVKh3JVp/miPgzTGAo7z
hrDIoXc+ih7orAMb19qol/2Ha8OZLa75LojJNRbZoCR5C+gM8C+spMLjFf9k3JVx
dajhtRUcR0zYhwsBS7qZ5Me0d6gRXD0ZiSbadMMxSw6KfKk3ePmPb9gX+MRTS63c
8mLzVYB/3fe/bkpq4RUwzUHvoZf+SUD7NzSQRQQMfvAHlxk11TVNxScYPtxXDyiy
3Cssl9gWrrWqQ/omuHipoH62J7h8KAYbr6oEIq+Czuenc3eCIBGBBfvCpuFOgckA
XXE4MlBasEU0MO66GrTCgMt9bAmSw3TrRP12+ZUFxYNtqWluRU8JWQ4FCCPcz9pg
MRBOgn4lTxDZG+I47OKNuSRjFEP94cdgxd3H/5BK7WHUz1tAGQ4BgepSXgmjzifF
T5FVTDTl3ZnWUVBXiHYtbOBgLiSIkbqGMCLtrBtFIeQ7RRTb3L+IE9R0UB0cJB3A
Xbf1lVkOcmrdu2h8A32aCwtr5S1fBF1unlG7imPmqJfpOMWa8yIF/KWVm29JAPq8
Lrsybb0z5gg8w7ZblEuB9zOW9M3l60DXuJO6l7g+deV6P96rv2unHS8UlvWiVWDy
9qfgAJizyy3kqM4lOwBH
-----END CERTIFICATE-----`
)

// newCertPool returns a [x509.CertPool], containing the custom Signal CA root certificates
func newCertPool() (*x509.CertPool, error) {
	certPool := netxlite.NewDefaultCertPool()
	signalCAByteSlice := [][]byte{
		[]byte(signalCA),
		[]byte(signalCANew),
	}
	for _, caBytes := range signalCAByteSlice {
		if !certPool.AppendCertsFromPEM(caBytes) {
			return nil, errors.New("AppendCertsFromPEM failed")
		}
	}
	return certPool, nil
}

// Config contains the signal experiment config.
type Config struct {
	// SignalCA is used to pass in a custom CA in testing
	SignalCA string
}

// TestKeys contains signal test keys.
type TestKeys struct {
	mu sync.Mutex

	Agent         string                   `json:"agent"`                // df-001-httpt
	SOCKSProxy    string                   `json:"socksproxy,omitempty"` // df-001-httpt
	Requests      []tracex.RequestEntry    `json:"requests"`             // df-001-httpt
	Queries       []tracex.DNSQueryEntry   `json:"queries"`              // df-002-dnst
	TCPConnect    []tracex.TCPConnectEntry `json:"tcp_connect"`          // df-005-tcpconnect
	TLSHandshakes []tracex.TLSHandshake    `json:"tls_handshakes"`       // df-006-tlshandshake
	NetworkEvents []tracex.NetworkEvent    `json:"network_events"`       // df-008-netevents

	SignalBackendStatus  string  `json:"signal_backend_status"`
	SignalBackendFailure *string `json:"signal_backend_failure"`
}

// NewTestKeys creates new signal TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		SignalBackendStatus:  "ok",
		SignalBackendFailure: nil,
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

// setFailure updates the TestKeys using the given error (goroutine safe).
func (tk *TestKeys) setFailure(err error) {
	defer tk.mu.Unlock()
	tk.mu.Lock()
	tk.SignalBackendStatus = "blocked"
	tk.SignalBackendFailure = tracex.NewFailure(err)
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

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	sess := args.Session
	measurement := args.Measurement
	tk := new(TestKeys)
	measurement.TestKeys = tk

	zeroTime := time.Now()
	certPool, err := newCertPool()
	if err != nil {
		return err // fundamental error, let's not submit
	}

	domains := []string{
		"textsecure-service.whispersystems.org",
		"storage.signal.org",
		"api.directory.signal.org",
		"cdn.signal.org",
		"cdn2.signal.org",
		"sfu.voip.signal.org",
		"uptime.signal.org",
	}

	// run measurements in parallel
	wg := &sync.WaitGroup{}
	for _, domain := range domains {
		wg.Add(1)
		go measureTarget(ctx, sess.Logger(), &atomic.Int64{}, zeroTime, tk, domain, certPool, wg)
	}
	wg.Wait()

	return nil
}

// measureTarget measures a signal backend domain
func measureTarget(
	ctx context.Context,
	logger model.Logger,
	idGen *atomic.Int64,
	zeroTime time.Time,
	tk *TestKeys,
	domain string,
	certPool *x509.CertPool,
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	// describe the DNS measurement input
	dnsInput := dslx.NewDomainToResolve(
		dslx.DomainName(domain),
		dslx.DNSLookupOptionIDGenerator(idGen),
		dslx.DNSLookupOptionLogger(logger),
		dslx.DNSLookupOptionZeroTime(zeroTime),
	)
	// construct getaddrinfo resolver
	lookup := dslx.DNSLookupGetaddrinfo()

	// run the DNS Lookup
	dnsResult := lookup.Apply(ctx, dnsInput)

	// extract and merge observations with the test keys
	tk.mergeObservations(dslx.ExtractObservations(dnsResult))

	// if the lookup has failed we set the error and return
	if dnsResult.Error != nil {
		tk.setFailure(dnsResult.Error)
		return
	}
	// for uptime.signal.org, we are only interested in the lookup, so we return here
	if domain == "uptime.signal.org" {
		return
	}

	// obtain a unique set of IP addresses w/o bogons inside it
	ipAddrs := dslx.NewAddressSet(dnsResult).RemoveBogons()

	// create the set of endpoints
	endpoints := ipAddrs.ToEndpoints(
		dslx.EndpointNetwork("tcp"),
		dslx.EndpointPort(443),
		dslx.EndpointOptionDomain(domain),
		dslx.EndpointOptionIDGenerator(idGen),
		dslx.EndpointOptionLogger(logger),
		dslx.EndpointOptionZeroTime(zeroTime),
	)

	// count the number of successful GET requests
	successes := dslx.Counter[*dslx.HTTPResponse]{}

	// create the established connections pool
	connpool := &dslx.ConnPool{}
	defer connpool.Close()

	// create function for the 443/tcp/tls/https measurement
	httpsFunction := dslx.Compose5(
		dslx.TCPConnect(connpool),
		dslx.TLSHandshake(
			connpool,
			dslx.TLSHandshakeOptionRootCAs(certPool),
		),
		dslx.HTTPTransportTLS(),
		dslx.HTTPRequest(),
		successes.Func(), // count number of times we arrive here
	)

	// run 443/tcp/tls/https measurement
	httpsResults := dslx.Map(
		ctx,
		dslx.Parallelism(2),
		httpsFunction,
		dslx.StreamList(endpoints...),
	)
	coll := dslx.Collect(httpsResults)

	// extract and merge observations with the test keys
	tk.mergeObservations(dslx.ExtractObservations(coll...))

	// if we saw successes, then this domain is not blocked
	// TODO: Success = at least one endpoint succeeds?
	if successes.Value() > 0 {
		return
	}

	// else we find the first error and store it in the test keys
	_, firstError := dslx.FirstErrorExcludingBrokenIPv6Errors(coll...)
	if firstError != nil {
		tk.setFailure(firstError)
		return
	}
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
	SignalBackendStatus  string  `json:"signal_backend_status"`
	SignalBackendFailure *string `json:"signal_backend_failure"`
	IsAnomaly            bool    `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return nil, errors.New("invalid test keys type")
	}
	sk.SignalBackendStatus = tk.SignalBackendStatus
	sk.SignalBackendFailure = tk.SignalBackendFailure
	sk.IsAnomaly = tk.SignalBackendStatus == "blocked"
	return sk, nil
}
