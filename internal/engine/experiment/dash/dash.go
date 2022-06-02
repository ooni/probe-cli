// Package dash implements the DASH network experiment.
//
// Spec: https://github.com/ooni/spec/blob/master/nettests/ts-021-dash.md
package dash

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
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	defaultTimeout = 120 * time.Second
	magicVersion   = "0.008000000"
	testName       = "dash"
	testVersion    = "0.13.0"
	totalStep      = 15
)

var (
	errServerBusy        = errors.New("dash: server busy; try again later")
	errHTTPRequestFailed = errors.New("dash: request failed")
)

// Config contains the experiment config.
type Config struct{}

// Simple contains the experiment total summary
type Simple struct {
	ConnectLatency  float64 `json:"connect_latency"`
	MedianBitrate   int64   `json:"median_bitrate"`
	MinPlayoutDelay float64 `json:"min_playout_delay"`
}

// ServerInfo contains information on the selected server
//
// This is currently an extension to the DASH specification
// until the data format of the new mlab locate is clear.
type ServerInfo struct {
	Hostname string `json:"hostname"`
	Site     string `json:"site,omitempty"`
}

// TestKeys contains the test keys
type TestKeys struct {
	Server       ServerInfo      `json:"server"`
	Simple       Simple          `json:"simple"`
	Failure      *string         `json:"failure"`
	ReceiverData []clientResults `json:"receiver_data"`
}

type runner struct {
	callbacks  model.ExperimentCallbacks
	httpClient *http.Client
	saver      *tracex.Saver
	sess       model.ExperimentSession
	tk         *TestKeys
}

func (r runner) HTTPClient() *http.Client {
	return r.httpClient
}

func (r runner) JSONMarshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (r runner) Logger() model.Logger {
	return r.sess.Logger()
}

func (r runner) NewHTTPRequest(meth, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequest(meth, url, body)
}

func (r runner) ReadAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	return netxlite.ReadAllContext(ctx, reader)
}

func (r runner) Scheme() string {
	return "https"
}

func (r runner) UserAgent() string {
	return r.sess.UserAgent()
}

func (r runner) loop(ctx context.Context, numIterations int64) error {
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
	negotiateResp, err := negotiate(ctx, fqdn, r)
	if err != nil {
		return err
	}
	if err := r.measure(ctx, fqdn, negotiateResp, numIterations); err != nil {
		return err
	}
	// TODO(bassosimone): it seems we're not saving the server data?
	err = collect(ctx, fqdn, negotiateResp.Authorization, r.tk.ReceiverData, r)
	if err != nil {
		return err
	}
	return r.tk.analyze()
}

func (r runner) measure(
	ctx context.Context, fqdn string, negotiateResp negotiateResponse,
	numIterations int64) error {
	// Note: according to a comment in MK sources 3000 kbit/s was the
	// minimum speed recommended by Netflix for SD quality in 2017.
	//
	// See: <https://help.netflix.com/en/node/306>.
	const initialBitrate = 3000
	current := clientResults{
		ElapsedTarget: 2,
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
	for current.Iteration < numIterations {
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
		current.Elapsed = result.elapsed
		current.Received = result.received
		current.RequestTicks = result.requestTicks
		current.Timestamp = result.timestamp
		current.ServerURL = result.serverURL
		// Read the events so far and possibly update our measurement
		// of the latest connect time. We should have one sample in most
		// cases, because the connection should be persistent.
		for _, ev := range r.saver.Read() {
			if _, ok := ev.(*tracex.EventConnectOperation); ok {
				connectTime = ev.Value().Duration.Seconds()
			}
		}
		current.ConnectTime = connectTime
		r.tk.ReceiverData = append(r.tk.ReceiverData, current)
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
	tk.Simple.MedianBitrate = int64(median)
	return err
}

func (r runner) do(ctx context.Context) error {
	defer r.callbacks.OnProgress(1, "streaming: done")
	const numIterations = 15
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
func (m Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	tk := new(TestKeys)
	measurement.TestKeys = tk
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
	r := runner{
		callbacks:  callbacks,
		httpClient: httpClient,
		saver:      saver,
		sess:       sess,
		tk:         tk,
	}
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
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
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
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
