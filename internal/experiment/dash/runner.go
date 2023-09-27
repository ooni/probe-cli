package dash

//
// runner: code that runs this experiment.
//

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// runnerConfig contains settings for running the dash experiment. This struct
// also implements [dependencies] thus allowing for unit testing of dash.
type runnerConfig struct {
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

var _ dependencies = &runnerConfig{}

// HTTPClient returns the configured HTTP client.
func (r *runnerConfig) HTTPClient() model.HTTPClient {
	return r.httpClient
}

// Logger returns the logger to use.
func (r *runnerConfig) Logger() model.Logger {
	return r.sess.Logger()
}

// NewHTTPRequestWithContext allows mocking the [http.NewRequestWithContext] function.
func (r *runnerConfig) NewHTTPRequestWithContext(
	ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	return http.NewRequestWithContext(ctx, method, url, body)
}

// UserAgent returns the user-agent to use.
func (r *runnerConfig) UserAgent() string {
	return r.sess.UserAgent()
}

// runnerRunAllPhases runs all the experiment phases.
func runnerRunAllPhases(ctx context.Context, r *runnerConfig, numIterations int64) error {
	// 1. locate the server with which to perform the measurement
	locateResult, err := locate(ctx, r)
	if err != nil {
		return err
	}
	runtimex.Assert(locateResult != nil, "nil locateResult")
	r.tk.Server = ServerInfo{
		Hostname: locateResult.Hostname,
		Site:     locateResult.Site,
	}
	hostname := locateResult.Hostname
	r.callbacks.OnProgress(0.0, fmt.Sprintf("streaming: server: %s", hostname))

	// 2. negotiate with the server and immediately bail in case
	// there is an error. Historically, the server could choose not
	// to allow us to perform a measurement and the Neubot client
	// would loop until given the authorization. Nowadays, the server
	// always admits us and the queuing is handled centrally by the
	// m-lab locate API.
	negotiateResp, err := negotiate(ctx, locateResult.NegotiateURL, r)
	if err != nil {
		return err
	}

	// 3. perform the measurement loop running for numIterations iterations.
	//
	// Implementation note: while we MUST use the NegotiateURL for negotiating such
	// that we consume m-lab's access token, we are free to use the BaseURL for
	// subsequent operations, since just negotiate is token aware.
	if err := runnerMeasure(ctx, r, locateResult.BaseURL, negotiateResp, numIterations); err != nil {
		return err
	}

	// 4. [collect] send our measurements to the server and receive
	// server-side measurements.
	//
	// Implementation note: we are not saving server-side measurements
	// because historically the interesting DASH measurement is the one
	// performed on the client side.
	err = collect(ctx, locateResult.BaseURL, negotiateResp.Authorization, r.tk.ReceiverData, r)
	if err != nil {
		return err
	}

	// 5. analyze the measurement results.
	return r.tk.analyze()
}

// runnerMeasure performs DASH measurements with the server. The numIterations
// parameter controls the total number of iterations we'll make.
func runnerMeasure(
	ctx context.Context,
	r *runnerConfig,
	baseURL string,
	negotiateResp negotiateResponse,
	numIterations int64,
) error {

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
			baseURL:       baseURL,
			begin:         begin,
			currentRate:   current.Rate,
			deps:          r,
			elapsedTarget: current.ElapsedTarget,
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
		// small refactoring away to produce all equal structures! We
		// added a WARNING in the TestKeys doc to defend against such a
		// dangerous refactoring, while waiting to make code more robust.
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

// runnerMain is the main function that runs the experiment.
func runnerMain(ctx context.Context, r *runnerConfig) error {
	defer r.callbacks.OnProgress(1, "streaming: done")
	err := runnerRunAllPhases(ctx, r, totalStep)
	if err != nil {
		s := err.Error()
		r.tk.Failure = &s
		// fallthrough
	}
	return err
}
