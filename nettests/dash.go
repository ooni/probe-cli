package nettests

import (
	"github.com/pkg/errors"
)

// Dash test implementation
type Dash struct {
}

// Run starts the test
func (d Dash) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dash")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// DashTestKeys for the test
// TODO: process 'receiver_data' to provide an array of performance for a chart.
type DashTestKeys struct {
	Latency   float64 `json:"connect_latency"`
	Bitrate   float64 `json:"median_bitrate"`
	Delay     float64 `json:"min_playout_delay"`
	IsAnomaly bool    `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (d Dash) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var err error

	testKeys := DashTestKeys{IsAnomaly: false}

	simple, ok := tk["simple"].(map[string]interface{})
	if !ok {
		return testKeys, errors.New("simple key is not of the expected type")
	}

	latency, ok := simple["connect_latency"].(float64)
	if !ok {
		err = errors.Wrap(err, "connect_latency is invalid")
	}
	testKeys.Latency = latency

	bitrate, ok := simple["median_bitrate"].(float64)
	if !ok {
		err = errors.Wrap(err, "median_bitrate is invalid")
	}
	testKeys.Bitrate = bitrate

	delay, ok := simple["min_playout_delay"].(float64)
	if !ok {
		err = errors.Wrap(err, "min_playout_delay is invalid")
	}
	testKeys.Delay = delay
	return testKeys, err
}

// LogSummary writes the summary to the standard output
func (d Dash) LogSummary(s string) error {
	return nil
}
