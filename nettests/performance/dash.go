package performance

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
)

// Dash test implementation
type Dash struct {
}

// Run starts the test
func (d Dash) Run(ctl *nettests.Controller) error {
	dash := mk.NewNettest("Dash")
	ctl.Init(dash)
	return dash.Run()
}

// DashTestKeys for the test
// TODO: process 'receiver_data' to provide an array of performance for a chart.
type DashTestKeys struct {
	Latency   float64 `json:"connect_latency"`
	Bitrate   int64   `json:"median_bitrate"`
	Delay     float64 `json:"min_playout_delay"`
	IsAnomaly bool    `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (d Dash) GetTestKeys(tk map[string]interface{}) interface{} {
	simple := tk["simple"].(map[string]interface{})

	return DashTestKeys{
		IsAnomaly: false,
		Latency:   simple["connect_latency"].(float64),
		Bitrate:   int64(simple["median_bitrate"].(float64)),
		Delay:     simple["min_playout_delay"].(float64),
	}
}

// LogSummary writes the summary to the standard output
func (d Dash) LogSummary(s string) error {
	return nil
}
