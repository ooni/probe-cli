package dash

//
// Implementation of [model.ExperimentMeasurer].
//

import (
	"context"
	"errors"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// Config contains the experiment config.
type Config struct{}

// Simple contains the experiment summary.
type Simple struct {
	ConnectLatency  float64 `json:"connect_latency"`
	MedianBitrate   int64   `json:"median_bitrate"`
	MinPlayoutDelay float64 `json:"min_playout_delay"`
}

// ServerInfo contains information on the selected server.
//
// This is currently an extension to the DASH specification
// until the data format of mlab locate v2 is finalized.
type ServerInfo struct {
	Hostname string `json:"hostname"`
	Site     string `json:"site,omitempty"`
}

// TestKeys contains the test keys.
type TestKeys struct {
	// ServerInfo contains information about the server we used.
	Server ServerInfo `json:"server"`

	// Simple contains simple summary statistics.
	Simple Simple `json:"simple"`

	// Failure is the failure that occurred.
	Failure *string `json:"failure"`

	// ReceiverData contains the results.
	//
	// WARNING: refactoring this field to become []*clientResults
	// would be disastrous because the measurement loop relies
	// on this slice being []clientResults to produce distinct results.
	ReceiverData []clientResults `json:"receiver_data"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	// unwrap arguments
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// create and set the test keys
	tk := &TestKeys{}
	measurement.TestKeys = tk

	// create a special purpose HTTP client for the measurement.
	saver := &tracex.Saver{}
	httpClient := &http.Client{
		Transport: netx.NewHTTPTransport(netx.Config{
			ContextByteCounting: true,
			// Implements shaping if the user builds using `-tags shaping`
			// See https://github.com/ooni/probe/issues/2112
			Dialer: netxlite.NewMaybeShapingDialer(netx.NewDialer(netx.Config{
				ContextByteCounting: true,
				Saver:               saver,
				Logger:              sess.Logger(),
			})),
			Logger: sess.Logger(),
		}),
	}
	defer httpClient.CloseIdleConnections()

	// configure the overall timeout for the experiment
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// create an instance of runner.
	r := &runner{
		callbacks:  callbacks,
		httpClient: httpClient,
		saver:      saver,
		sess:       sess,
		tk:         tk,
	}

	// run the experiment.
	//
	// Implementation note: we ignore the return value of r.do rather than
	// returning it to the caller. We do that because returning an error means
	// the measurement failed for some fundamental reason (e.g., the input
	// is an URL that you cannot parse). For DASH, this case will never happen
	// because there is no input, so always returning nil is fine here.
	_ = runnerMain(ctx, r)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	Latency   float64 `json:"connect_latency"`
	Bitrate   float64 `json:"median_bitrate"`
	Delay     float64 `json:"min_playout_delay"`
	IsAnomaly bool    `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (any, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.Latency = tk.Simple.ConnectLatency
	sk.Bitrate = float64(tk.Simple.MedianBitrate)
	sk.Delay = tk.Simple.MinPlayoutDelay
	return sk, nil
}
