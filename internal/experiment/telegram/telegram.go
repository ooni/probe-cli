// Package telegram contains the Telegram network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-020-telegram.md.
package telegram

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/richerinput"
)

// NewRicherInputExperiment constructs a new [model.RicherInputExperiment] instance.
func NewRicherInputExperiment(cbs model.ExperimentCallbacks, sess model.RicherInputSession) model.RicherInputExperiment {
	return richerinput.NewExperiment(
		cbs,
		sess,
		testName,
		testVersion,
		func(config Config) model.ExperimentMeasurer {
			return NewExperimentMeasurer(config)
		},
	)
}

const (
	testName    = "telegram"
	testVersion = "0.3.1"
)

// Config contains the telegram experiment config.
type Config struct{}

// TestKeys contains telegram test keys.
type TestKeys struct {
	urlgetter.TestKeys
	TelegramHTTPBlocking bool    `json:"telegram_http_blocking"`
	TelegramTCPBlocking  bool    `json:"telegram_tcp_blocking"`
	TelegramWebFailure   *string `json:"telegram_web_failure"`
	TelegramWebStatus    string  `json:"telegram_web_status"`
}

// NewTestKeys creates new telegram TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		TelegramHTTPBlocking: true,
		TelegramTCPBlocking:  true,
		TelegramWebFailure:   nil,
		TelegramWebStatus:    "ok",
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
	// then process access points
	if v.Input.Config.Method != "GET" {
		if v.TestKeys.Failure == nil {
			tk.TelegramHTTPBlocking = false
			tk.TelegramTCPBlocking = false
			return // found successful access point connection
		}
		if v.TestKeys.FailedOperation == nil || *v.TestKeys.FailedOperation != netxlite.ConnectOperation {
			tk.TelegramTCPBlocking = false
		}
		return
	}
	if v.TestKeys.Failure != nil {
		tk.TelegramWebStatus = "blocked"
		tk.TelegramWebFailure = v.TestKeys.Failure
		return
	}
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

// DatacenterIPAddrs contains the list of Telegram data centers IP addresses to measure.
var DatacenterIPAddrs = []string{
	"149.154.175.50",
	"149.154.167.51",
	"149.154.175.100",
	"149.154.167.91",
	"149.154.171.5",
	"95.161.76.100",
}

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	urlgetter.RegisterExtensions(measurement)
	inputs := []urlgetter.MultiInput{
		// Here we need to provide the method explicitly. See
		// https://github.com/ooni/probe-engine/issues/827.
		{Target: "https://web.telegram.org/", Config: urlgetter.Config{
			Method: "GET",
		}},
	}

	// We need to measure each address twice. Once using port 80 and once using port 443. In both
	// cases, the protocol MUST be HTTP. The DCs do not support access on port 443 using TLS.
	for _, dc := range DatacenterIPAddrs {
		inputs = append(inputs, urlgetter.MultiInput{Target: "http://" + dc + "/", Config: urlgetter.Config{Method: "POST"}})
		inputs = append(inputs, urlgetter.MultiInput{Target: "http://" + dc + ":443/", Config: urlgetter.Config{Method: "POST"}})
	}

	multi := urlgetter.Multi{Begin: time.Now(), Getter: m.Getter, Session: sess}
	testkeys := NewTestKeys()
	testkeys.Agent = "redirect"
	measurement.TestKeys = testkeys
	for entry := range multi.Collect(ctx, inputs, "telegram", callbacks) {
		testkeys.Update(entry)
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

var _ model.MeasurementSummaryKeysProvider = &TestKeys{}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	HTTPBlocking bool `json:"telegram_http_blocking"`
	TCPBlocking  bool `json:"telegram_tcp_blocking"`
	WebBlocking  bool `json:"telegram_web_blocking"`
	IsAnomaly    bool `json:"-"`
}

// MeasurementSummaryKeys implements model.MeasurementSummaryKeysProvider.
func (tk *TestKeys) MeasurementSummaryKeys() model.MeasurementSummaryKeys {
	sk := &SummaryKeys{IsAnomaly: false}
	tcpBlocking := tk.TelegramTCPBlocking
	httpBlocking := tk.TelegramHTTPBlocking
	webBlocking := tk.TelegramWebFailure != nil
	sk.TCPBlocking = tcpBlocking
	sk.HTTPBlocking = httpBlocking
	sk.WebBlocking = webBlocking
	sk.IsAnomaly = webBlocking || httpBlocking || tcpBlocking
	return sk
}

// Anomaly implements model.MeasurementSummaryKeys.
func (sk *SummaryKeys) Anomaly() bool {
	return sk.IsAnomaly
}
