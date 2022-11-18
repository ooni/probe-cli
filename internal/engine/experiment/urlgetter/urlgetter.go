// Package urlgetter implements a nettest that fetches a URL.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-027-urlgetter.md.
//
// This package is now frozen. Please, use measurexlite for new code. New
// network experiments should not depend on this package. Please see
// https://github.com/ooni/probe-cli/blob/master/docs/design/dd-003-step-by-step.md
// for details about this.
package urlgetter

import (
	"context"
	"crypto/x509"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName    = "urlgetter"
	testVersion = "0.2.0"
)

// Config contains the experiment's configuration.
type Config struct {
	// not settable from command line
	CertPool *x509.CertPool
	Timeout  time.Duration
	Dialer   model.Dialer

	// settable from command line
	DNSCache          string `ooni:"Add 'DOMAIN IP...' to cache"`
	DNSHTTPHost       string `ooni:"Force using specific HTTP Host header for DNS requests"`
	DNSTLSServerName  string `ooni:"Force TLS to using a specific SNI for encrypted DNS requests"`
	DNSTLSVersion     string `ooni:"Force specific TLS version used for DoT/DoH (e.g. 'TLSv1.3')"`
	FailOnHTTPError   bool   `ooni:"Fail HTTP request if status code is 400 or above"`
	HTTP3Enabled      bool   `ooni:"use http3 instead of http/1.1 or http2"`
	HTTPHost          string `ooni:"Force using specific HTTP Host header"`
	Method            string `ooni:"Force HTTP method different than GET"`
	NoFollowRedirects bool   `ooni:"Disable following redirects"`
	NoTLSVerify       bool   `ooni:"Disable TLS verification"`
	RejectDNSBogons   bool   `ooni:"Fail DNS lookup if response contains bogons"`
	ResolverURL       string `ooni:"URL describing the resolver to use"`
	TLSServerName     string `ooni:"Force TLS to using a specific SNI in Client Hello"`
	TLSVersion        string `ooni:"Force specific TLS version (e.g. 'TLSv1.3')"`
	Tunnel            string `ooni:"Run experiment over a tunnel, e.g. psiphon"`
	UserAgent         string `ooni:"Use the specified User-Agent"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// The following fields are part of the typical JSON emitted by OONI.
	Agent           string                   `json:"agent"`
	BootstrapTime   float64                  `json:"bootstrap_time,omitempty"`
	DNSCache        []string                 `json:"dns_cache,omitempty"`
	FailedOperation *string                  `json:"failed_operation"`
	Failure         *string                  `json:"failure"`
	NetworkEvents   []tracex.NetworkEvent    `json:"network_events"`
	Queries         []tracex.DNSQueryEntry   `json:"queries"`
	Requests        []tracex.RequestEntry    `json:"requests"`
	SOCKSProxy      string                   `json:"socksproxy,omitempty"`
	TCPConnect      []tracex.TCPConnectEntry `json:"tcp_connect"`
	TLSHandshakes   []tracex.TLSHandshake    `json:"tls_handshakes"`
	Tunnel          string                   `json:"tunnel,omitempty"`

	// The following fields are not serialised but are useful to simplify
	// analysing the measurements in telegram, whatsapp, etc.
	HTTPResponseStatus    int64    `json:"-"`
	HTTPResponseBody      string   `json:"-"`
	HTTPResponseLocations []string `json:"-"`
}

// RegisterExtensions registers the extensions used by the urlgetter
// experiment into the provided measurement.
func RegisterExtensions(m *model.Measurement) {
	tracex.ExtHTTP.AddTo(m)
	tracex.ExtDNS.AddTo(m)
	tracex.ExtNetevents.AddTo(m)
	tracex.ExtTCPConnect.AddTo(m)
	tracex.ExtTLSHandshake.AddTo(m)
	tracex.ExtTunnel.AddTo(m)
}

// Measurer performs the measurement.
type Measurer struct {
	Config
}

// ExperimentName implements model.ExperimentSession.ExperimentName
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentSession.ExperimentVersion
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements model.ExperimentSession.Run
func (m Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	// When using the urlgetter experiment directly, there is a nonconfigurable
	// default timeout that applies. When urlgetter is used as a library, it's
	// instead the responsibility of the user of urlgetter to set timeouts. Note
	// that this code is indeed only called when using urlgetter directly.
	if m.Config.Timeout <= 0 {
		m.Config.Timeout = 45 * time.Second
	}
	RegisterExtensions(measurement)
	g := Getter{
		Config:  m.Config,
		Session: sess,
		Target:  string(measurement.Input),
	}
	tk, _ := g.Get(ctx) // ignore error since we have the testkeys and we wanna submit them
	measurement.TestKeys = &tk
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
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
