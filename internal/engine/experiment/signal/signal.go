// Package signal contains the Signal network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-029-signal.md.
package signal

import (
	"context"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "signal"
	testVersion = "0.2.1"

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

	signalCANew = `-----BEGIN CERTIFICATE-----
MIIEjDCCAnSgAwIBAgITV+dgmSk1+75Wwn/Mjz8f+gQ9qTANBgkqhkiG9w0BAQsF
ADB1MQswCQYDVQQGEwJVUzETMBEGA1UECBMKQ2FsaWZvcm5pYTEWMBQGA1UEBxMN
TW91bnRhaW4gVmlldzEeMBwGA1UEChMVU2lnbmFsIE1lc3NlbmdlciwgTExDMRkw
FwYDVQQDExBTaWduYWwgTWVzc2VuZ2VyMB4XDTIyMDgyMzE2NTIxMVoXDTIzMDky
MzIyNDA1NlowADCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBALf8th0A
N5TFsvvdfaSP1WyCMn5Ql81IF5D0pXrdE9fGDz5AaeAbCazxXU8tnjZiUr4a/BGD
h3ZxORHXJ2SA3HA2UFG+qHik59QNGkY4Jv4emTM5QLw0fcsGRgJnzb7A60LRoxGs
17jxD1zyVl/SXn/Ql3cvBrHjxPzJ6NcQG4Pek7YieH2xiMP794QUu0XJYlBx0uvx
xOI3qpw5c6oNORGY8hlwWzbv+sqvShXhteOlkzluKtIqpL8+NV206JIqLkaKFjB7
To14TSFF3tYxxsHYwDhRKPatqYpbebx3iCo0H33dL0gjoUtdvRgsdHqnUQXSqoRH
cUYCIPs3FivKNrcCAwEAAaOBiTCBhjATBgNVHSUEDDAKBggrBgEFBQcDATAMBgNV
HRMBAf8EAjAAMB0GA1UdDgQWBBSidZq+TLJkcDuNV5j1KbOm/l+dhjAfBgNVHSME
GDAWgBS180vG5dZL0OWAa4xQw2dbvLHzcTAhBgNVHREBAf8EFzAVghNzZnUudm9p
cC5zaWduYWwub3JnMA0GCSqGSIb3DQEBCwUAA4ICAQCDchlftHXUm3sFWL86GKUs
w7nxOiJDZYR+xIVGbsUarBolEsZZkYjTDB427ZjgBS+Nfhhbrw4k2LMarkxf2TQX
aelPHRa5xNPVfkrN8xw4fv/8TLE9GSjKlrNJm1EoTZL5CYWQU+qe4CuKfAJU6h8l
xIkcik61aCeNLQoaI1L3V8tPXmmqMWpsnZmFg6YLGeMTLs4skdFqgLOnx9EF2jgO
7EAJ9HcrgSPirQeuDJKhamaLtQiqIQR8L3H4YG1FDiuOeto6f1LRCIqjH1Mye1BM
33Qg/VilLQIWp8+C4GJZ0+LO1cfatNh8tkDbrwMzUeA1nLEZHMlgXE05z00euNlQ
0+evTmJzWRKJHugPnA3vvdzy4lbYvYWaXs8pACrVpESui8I+v6jdH814lOxpDwNH
bPrxfOxhIxfFiVttCl3AQZBLJM6M0ty6/Q7bYsdNT23jKMl0AmDhj9qn/7dzYcVi
vI0XKaaJl4ov3IDbuMe0oZWhoLwzPuWxxkWDjTb8ngDnWZT1o5dAR9fltr38m42N
uA/SkxghiAMmvkC8nhEJ7yT2hme+rozPZSp1SSEDViDkA4KnnQpMcNiotCQpNOe7
YfA9uSnjHjZloRTPUgtkKQ3u8ZZprFQlS2jDE18BRGdh24V5OsCbMvFPtrEsjG4H
5xvkiIV0FpbMk4Gj8I4Hbw==
-----END CERTIFICATE-----`
)

// Config contains the signal experiment config.
type Config struct {
	// SignalCA is used to pass in a custom CA in testing
	SignalCA string
}

// TestKeys contains signal test keys.
type TestKeys struct {
	urlgetter.TestKeys
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

// Update updates the TestKeys using the given MultiOutput result.
func (tk *TestKeys) Update(v urlgetter.MultiOutput) {
	// update the easy to update entries first
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, v.TestKeys.Queries...)
	tk.Requests = append(tk.Requests, v.TestKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, v.TestKeys.TLSHandshakes...)
	// Ignore the result of the uptime DNS lookup
	if v.Input.Target == "dnslookup://uptime.signal.org" {
		return
	}
	if v.TestKeys.Failure != nil {
		tk.SignalBackendStatus = "blocked"
		tk.SignalBackendFailure = v.TestKeys.Failure
		return
	}
	return
}

// Measurer performs the measurement
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config

	// Getter is an optional getter to be used for testing.
	Getter urlgetter.MultiGetter
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
func (m Measurer) Run(ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	urlgetter.RegisterExtensions(measurement)

	certPool := netxlite.NewDefaultCertPool()
	signalCAByteSlice := [][]byte{
		[]byte(signalCA),
		[]byte(signalCANew),
	}
	if m.Config.SignalCA != "" {
		signalCAByteSlice = [][]byte{[]byte(m.Config.SignalCA)}
	}
	for _, caBytes := range signalCAByteSlice {
		if !certPool.AppendCertsFromPEM(caBytes) {
			return errors.New("AppendCertsFromPEM failed")
		}
	}

	inputs := []urlgetter.MultiInput{
		// Here we need to provide the method explicitly. See
		// https://github.com/ooni/probe-engine/issues/827.
		{Target: "https://textsecure-service.whispersystems.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "https://storage.signal.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "https://api.directory.signal.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "https://cdn.signal.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "https://cdn2.signal.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "https://sfu.voip.signal.org/", Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: false,
			CertPool:        certPool,
		}},
		{Target: "dnslookup://uptime.signal.org"},
	}
	multi := urlgetter.Multi{Begin: time.Now(), Getter: m.Getter, Session: sess}
	testkeys := NewTestKeys()
	testkeys.Agent = "redirect"
	measurement.TestKeys = testkeys
	for entry := range multi.Collect(ctx, inputs, "signal", callbacks) {
		testkeys.Update(entry)
	}
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
