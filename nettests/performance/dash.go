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

// DashSummary for the test
// TODO: process 'receiver_data' to provide an array of performance for a chart.
type DashSummary struct {
	Latency float64
	Bitrate int64
	Delay   float64
}

// Summary generates a summary for a test run
func (d Dash) Summary(tk map[string]interface{}) interface{} {
	simple := tk["simple"].(map[string]interface{})

	return DashSummary{
		Latency: simple["connect_latency"].(float64),
		Bitrate: int64(simple["median_bitrate"].(float64)),
		Delay:   simple["min_playout_delay"].(float64),
	}
}

// LogSummary writes the summary to the standard output
func (d Dash) LogSummary(s string) error {
	return nil
}
