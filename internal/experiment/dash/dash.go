package dash

//
// Top-level experiment implementation
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/legacy/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	// defaultTimeout is the default timeout for the whole experiment.
	defaultTimeout = 120 * time.Second

	// magicVersion encodes the version number of the tool we are
	// using according to the format originally used by Neubot. We
	// used "0.007000000" for Measurement Kit, which mapped to Neubot
	// v0.7.0. OONI pretends to be Neubot v0.8.0.
	magicVersion = "0.008000000"

	// testName is the name of the experiment.
	testName = "dash"

	// testVersion is the version of the experiment.
	testVersion = "0.14.0"

	// totalStep is the total number of steps we should run
	// during the download experiment.
	totalStep = 15
)

var (
	// errServerBusy is the error returned when the DASH server is busy.
	errServerBusy = errors.New("dash: server busy; try again later")

	// errHTTPRequest failed is the error returned when an HTTP request fails.
	errHTTPRequestFailed = errors.New("dash: request failed")
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

// runner runs the experiment.
type runner struct {
	// callbacks contains the callbacks for emitting progress.
	callbacks model.ExperimentCallbacks

	// httpClient is the HTTP client we're using.
	httpClient model.HTTPClient

	// saver is MUTABLE and is used to save the connect time of connections,
	// which is part of the DASH measurement results.
	saver *tracex.Saver

	// sess is the measurement session.
	sess model.ExperimentSession

	// tk contains the MUTABLE test keys.
	tk *TestKeys
}

// HTTPClient returns the configured HTTP client.
func (r runner) HTTPClient() model.HTTPClient {
	return r.httpClient
}

// JSONMarshal allows mocking the [json.Marshal] function.
func (r runner) JSONMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

// Logger returns the logger to use.
func (r runner) Logger() model.Logger {
	return r.sess.Logger()
}

// NewHTTPRequest allows mocking the [http.NewRequest] function.
func (r runner) NewHTTPRequest(meth, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(meth, url, body)
}

// RealAllContext allows mocking the [netxlite.ReadAllContext] function.
func (r runner) ReadAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	return netxlite.ReadAllContext(ctx, reader)
}

// Scheme returns the URL scheme to use.
func (r runner) Scheme() string {
	return "https"
}

// UserAgent returns the user-agent to use.
func (r runner) UserAgent() string {
	return r.sess.UserAgent()
}

// loop (probably a misnomer) is the main function of the experiment, where
// we perform in sequence all the dash experiment's phases.
func (r runner) loop(ctx context.Context, numIterations int64) error {
	// 1. locate the server with which to perform the measurement
	locateResult, err := locate(ctx, r)
	if err != nil {
		return err
	}
	r.tk.Server = ServerInfo{
		Hostname: locateResult.FQDN,
		Site:     locateResult.Site,
	}
	fqdn := locateResult.FQDN
	r.callbacks.OnProgress(0.0, fmt.Sprintf("streaming: server: %s", fqdn))

	// 2. negotiate with the server and immediately bail in case
	// there is an error. Historically, the server could choose not
	// to allow us to perform a measurement and the Neubot client
	// would loop until given the authorization. Nowadays, the server
	// always admits us and the queuing is handled centrally by the
	// m-lab locate API.
	negotiateResp, err := negotiate(ctx, fqdn, r)
	if err != nil {
		return err
	}

	// 3. perform the measurement loop running for numIterations iterations.
	if err := r.measure(ctx, fqdn, negotiateResp, numIterations); err != nil {
		return err
	}

	// 4. [collect] send our measurements to the server and receive
	// server-side measurements.
	//
	// Implementation note: we are not saving server-side measurements
	// because historically the interesting DASH measurement is the one
	// performed on the client side.
	err = collect(ctx, fqdn, negotiateResp.Authorization, r.tk.ReceiverData, r)
	if err != nil {
		return err
	}

	// 5. analyze the measurement results.
	return r.tk.analyze()
}

// measure performs DASH measurements with the server. The numIterations
// parameter controls the total number of iterations we'll make.
func (r runner) measure(
	ctx context.Context, fqdn string, negotiateResp negotiateResponse,
	numIterations int64) error {

	// 1. fill the initial client results.
	//
	// Note: according to a comment in MK sources 3000 kbit/s was the
	// minimum speed recommended by Netflix for SD quality in 2017.
	//
	// See: <https://help.netflix.com/en/node/306>.
	const initialBitrate = 3000
	current := clientResults{
		ElapsedTarget: 2, // we expect the download to run for two seconds.
		Platform:      runtime.GOOS,
		Rate:          initialBitrate,
		RealAddress:   negotiateResp.RealAddress,
		Version:       magicVersion,
	}

	var (
		begin       = time.Now()
		connectTime float64
		total       int64
	)

	// 2. perform all the iterations
	for current.Iteration < numIterations {

		// 2.1. attempt do download a chunk from the server.
		result, err := download(ctx, downloadConfig{
			authorization: negotiateResp.Authorization,
			begin:         begin,
			currentRate:   current.Rate,
			deps:          r,
			elapsedTarget: current.ElapsedTarget,
			fqdn:          fqdn,
		})
		if err != nil {
			// Implementation note: ndt7 controls the connection much
			// more than us and it can tell whether an error occurs when
			// connecting or later. We cannot say that very precisely
			// because, in principle, we may reconnect. So we always
			// return error here. This comment is being introduced so
			// that we don't do https://github.com/ooni/probe-engine/pull/526
			// again, because that isn't accurate.
			return err
		}

		// 2.2. fill the current measurement structure.
		//
		// TODO(bassosimone): we should create a new structure in each
		// loop rather than overwriting the same structure and relying on
		// copying to produce distinct structures. The current code is a
		// small refactoring away to produce all equal structures!
		current.Elapsed = result.elapsed
		current.Received = result.received
		current.RequestTicks = result.requestTicks
		current.Timestamp = result.timestamp
		current.ServerURL = result.serverURL

		// 2.3. Read the events so far and possibly update our measurement
		// of the latest connect time. We should have one sample in most
		// cases, because the connection should be persistent.
		for _, ev := range r.saver.Read() {
			if _, ok := ev.(*tracex.EventConnectOperation); ok {
				connectTime = ev.Value().Duration.Seconds()
			}
		}
		current.ConnectTime = connectTime

		// 2.4. save the current measurement.
		//
		// TODO(bassosimone): see the above comment about refactoring.
		r.tk.ReceiverData = append(r.tk.ReceiverData, current)

		// 2.5. update the state variables and emit progress.
		total += current.Received
		avgspeed := 8 * float64(total) / time.Since(begin).Seconds()
		percentage := float64(current.Iteration) / float64(numIterations)
		message := fmt.Sprintf("streaming: speed: %s", humanize.SI(avgspeed, "bit/s"))
		r.callbacks.OnProgress(percentage, message)
		current.Iteration++
		speed := float64(current.Received) / float64(current.Elapsed)
		speed *= 8.0    // to bits per second
		speed /= 1000.0 // to kbit/s
		current.Rate = int64(speed)
	}

	return nil
}

// analyze analyzes the measurement results and fills tk.Simple.
func (tk *TestKeys) analyze() error {
	var (
		rates          []float64
		frameReadyTime float64
		playTime       float64
	)
	for _, results := range tk.ReceiverData {
		rates = append(rates, float64(results.Rate))

		// Same in all samples if we're using a single connection
		tk.Simple.ConnectLatency = results.ConnectTime

		// Rationale: first segment plays when it arrives. Subsequent segments
		// would play in ElapsedTarget seconds. However, will play when they
		// arrive. Stall is the time we need to wait for a frame to arrive with
		// the video stopped and the spinning icon.
		frameReadyTime += results.Elapsed
		if playTime == 0.0 {
			playTime += frameReadyTime
		} else {
			playTime += float64(results.ElapsedTarget)
		}
		stall := frameReadyTime - playTime
		if stall > tk.Simple.MinPlayoutDelay {
			tk.Simple.MinPlayoutDelay = stall
		}
	}

	median, err := stats.Median(rates)
	tk.Simple.MedianBitrate = int64(median) // on error we compute the median of an empty array
	return err
}

// do runs the experiment.
func (r runner) do(ctx context.Context) error {
	defer r.callbacks.OnProgress(1, "streaming: done")
	const numIterations = totalStep
	err := r.loop(ctx, numIterations)
	if err != nil {
		s := err.Error()
		r.tk.Failure = &s
		// fallthrough
	}
	return err
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements model.ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
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
	r := runner{
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
	_ = r.do(ctx)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{config: config}
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
