// Package ndt7 contains the ndt7 network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-022-ndt.md
package ndt7

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/mlablocatev2"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/humanize"
)

const (
	testName    = "ndt"
	testVersion = "0.8.0"
)

// Config contains the experiment settings
type Config struct {
	noDownload bool
	noUpload   bool
}

// Summary is the measurement summary
type Summary struct {
	AvgRTT         float64 `json:"avg_rtt"`         // Average RTT [ms]
	Download       float64 `json:"download"`        // download speed [kbit/s]
	MSS            int64   `json:"mss"`             // MSS
	MaxRTT         float64 `json:"max_rtt"`         // Max AvgRTT sample seen [ms]
	MinRTT         float64 `json:"min_rtt"`         // Min RTT according to kernel [ms]
	Ping           float64 `json:"ping"`            // Equivalent to MinRTT [ms]
	RetransmitRate float64 `json:"retransmit_rate"` // bytes_retrans/bytes_sent [0..1]
	Upload         float64 `json:"upload"`          // upload speed [kbit/s]
}

// ServerInfo contains information on the selected server
//
// Site is currently an extension to the NDT specification
// until the data format of the new mlab locate is clear.
type ServerInfo struct {
	Hostname string `json:"hostname"`
	Site     string `json:"site,omitempty"`
}

// TestKeys contains the test keys
type TestKeys struct {
	// Download contains download results
	Download []Measurement `json:"download"`

	// Failure is the failure string
	Failure *string `json:"failure"`

	// Protocol contains the version of the ndt protocol
	Protocol int64 `json:"protocol"`

	// Server contains information on the selected server
	Server ServerInfo `json:"server"`

	// Summary contains the measurement summary
	Summary Summary `json:"summary"`

	// Upload contains upload results
	Upload []Measurement `json:"upload"`
}

// Measurer performs the measurement.
type Measurer struct {
	config          Config
	jsonUnmarshal   func(data []byte, v interface{}) error
	preDownloadHook func()
	preUploadHook   func()
}

func (m *Measurer) discover(
	ctx context.Context, sess model.ExperimentSession) (mlablocatev2.NDT7Result, error) {
	httpClient := &http.Client{
		Transport: netx.NewHTTPTransport(netx.Config{
			Logger: sess.Logger(),
		}),
	}
	defer httpClient.CloseIdleConnections()
	client := mlablocatev2.NewClient(httpClient, sess.Logger(), sess.UserAgent())
	out, err := client.QueryNDT7(ctx)
	if err != nil {
		return mlablocatev2.NDT7Result{}, err
	}
	return out[0], nil // same as with locate services v1
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) doDownload(
	ctx context.Context, sess model.ExperimentSession,
	callbacks model.ExperimentCallbacks, tk *TestKeys,
	URL string,
) error {
	if m.config.noDownload {
		return nil // useful to make tests faster
	}
	conn, err := newDialManager(URL,
		sess.Logger(), sess.UserAgent()).dialDownload(ctx)
	if err != nil {
		return err
	}
	defer callbacks.OnProgress(0.5, " download: done")
	defer conn.Close()
	mgr := newDownloadManager(
		conn,
		func(timediff time.Duration, count int64) {
			elapsed := timediff.Seconds()
			// The percentage of completion of download goes from 0 to
			// 50% of the whole experiment, hence the `/2.0`.
			percentage := elapsed / paramMaxRuntimeUpperBound / 2.0
			speed := float64(count) * 8.0 / elapsed
			message := fmt.Sprintf(" download: speed %s", humanize.SI(
				float64(speed), "bit/s"))
			tk.Summary.Download = speed / 1e03 /* bit/s => kbit/s */
			callbacks.OnProgress(percentage, message)
			tk.Download = append(tk.Download, Measurement{
				AppInfo: &AppInfo{
					ElapsedTime: int64(timediff / time.Microsecond),
					NumBytes:    count,
				},
				Origin: "client",
				Test:   "download",
			})
		},
		func(data []byte) error {
			sess.Logger().Debugf("%s", string(data))
			var measurement Measurement
			if err := m.jsonUnmarshal(data, &measurement); err != nil {
				return err
			}
			if measurement.TCPInfo != nil {
				rtt := float64(measurement.TCPInfo.RTT) / 1e03 /* us => ms */
				tk.Summary.AvgRTT = rtt
				tk.Summary.MSS = int64(measurement.TCPInfo.AdvMSS)
				if tk.Summary.MaxRTT < rtt {
					tk.Summary.MaxRTT = rtt
				}
				tk.Summary.MinRTT = float64(measurement.TCPInfo.MinRTT) / 1e03 /* us => ms */
				tk.Summary.Ping = tk.Summary.MinRTT
				if measurement.TCPInfo.BytesSent > 0 {
					tk.Summary.RetransmitRate = (float64(measurement.TCPInfo.BytesRetrans) /
						float64(measurement.TCPInfo.BytesSent))
				}
				measurement.BBRInfo = nil        // don't encourage people to use it
				measurement.ConnectionInfo = nil // do we need to save it?
				measurement.Origin = "server"
				measurement.Test = "download"
				tk.Download = append(tk.Download, measurement)
			}
			return nil
		},
	)
	if err := mgr.run(ctx); err != nil && err.Error() != "generic_timeout_error" {
		sess.Logger().Warnf("download: %s", err)
	}
	return nil // failure is only when we cannot connect
}

func (m *Measurer) doUpload(
	ctx context.Context, sess model.ExperimentSession,
	callbacks model.ExperimentCallbacks, tk *TestKeys,
	URL string,
) error {
	if m.config.noUpload {
		return nil // useful to make tests faster
	}
	conn, err := newDialManager(URL,
		sess.Logger(), sess.UserAgent()).dialUpload(ctx)
	if err != nil {
		return err
	}
	defer callbacks.OnProgress(1, "   upload: done")
	defer conn.Close()
	mgr := newUploadManager(
		conn,
		func(timediff time.Duration, count int64) {
			elapsed := timediff.Seconds()
			// The percentage of completion of upload goes from 50% to 100% of
			// the whole experiment, hence `0.5 +` and `/2.0`.
			percentage := 0.5 + elapsed/paramMaxRuntimeUpperBound/2.0
			speed := float64(count) * 8.0 / elapsed
			message := fmt.Sprintf("   upload: speed %s", humanize.SI(
				float64(speed), "bit/s"))
			tk.Summary.Upload = speed / 1e03 /* bit/s => kbit/s */
			callbacks.OnProgress(percentage, message)
			tk.Upload = append(tk.Upload, Measurement{
				AppInfo: &AppInfo{
					ElapsedTime: int64(timediff / time.Microsecond),
					NumBytes:    count,
				},
				Origin: "client",
				Test:   "upload",
			})
		},
	)
	if err := mgr.run(ctx); err != nil && err.Error() != "generic_timeout_error" {
		sess.Logger().Warnf("upload: %s", err)
	}
	return nil // failure is only when we cannot connect
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	tk := new(TestKeys)
	tk.Protocol = 7
	measurement.TestKeys = tk
	locateResult, err := m.discover(ctx, sess)
	if err != nil {
		tk.Failure = failureFromError(err)
		return err
	}
	tk.Server = ServerInfo{
		Hostname: locateResult.Hostname,
		Site:     locateResult.Site,
	}
	callbacks.OnProgress(0, fmt.Sprintf(" download: url: %s", locateResult.WSSDownloadURL))
	if m.preDownloadHook != nil {
		m.preDownloadHook()
	}
	if err := m.doDownload(ctx, sess, callbacks, tk, locateResult.WSSDownloadURL); err != nil {
		tk.Failure = failureFromError(err)
		return err
	}
	callbacks.OnProgress(0.5, fmt.Sprintf("   upload: url: %s", locateResult.WSSUploadURL))
	if m.preUploadHook != nil {
		m.preUploadHook()
	}
	if err := m.doUpload(ctx, sess, callbacks, tk, locateResult.WSSUploadURL); err != nil {
		tk.Failure = failureFromError(err)
		return err
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config, jsonUnmarshal: json.Unmarshal}
}

func failureFromError(err error) (failure *string) {
	if err != nil {
		s := err.Error()
		failure = &s
	}
	return
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	Upload         float64 `json:"upload"`
	Download       float64 `json:"download"`
	Ping           float64 `json:"ping"`
	MaxRTT         float64 `json:"max_rtt"`
	AvgRTT         float64 `json:"avg_rtt"`
	MinRTT         float64 `json:"min_rtt"`
	MSS            float64 `json:"mss"`
	RetransmitRate float64 `json:"retransmit_rate"`
	IsAnomaly      bool    `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.Upload = tk.Summary.Upload
	sk.Download = tk.Summary.Download
	sk.Ping = tk.Summary.Ping
	sk.MaxRTT = tk.Summary.MaxRTT
	sk.AvgRTT = tk.Summary.AvgRTT
	sk.MinRTT = tk.Summary.MinRTT
	sk.MSS = float64(tk.Summary.MSS)
	sk.RetransmitRate = tk.Summary.RetransmitRate
	return sk, nil
}
