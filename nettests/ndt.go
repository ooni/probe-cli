package nettests

import (
	"github.com/pkg/errors"
)

// NDT test implementation. We use v7 of NDT since 2020-03-12.
type NDT struct {
}

// Run starts the test
func (n NDT) Run(ctl *Controller) error {
	// Since 2020-03-18 probe-engine exports v7 as "ndt".
	builder, err := ctl.Ctx.Session.NewExperimentBuilder("ndt")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// NDTTestKeys for the test
type NDTTestKeys struct {
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

// GetTestKeys generates a summary for a test run
func (n NDT) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var err error
	testKeys := NDTTestKeys{IsAnomaly: false}

	summary, ok := tk["summary"].(map[string]interface{})
	if !ok {
		return testKeys, errors.New("summary key is invalid")
	}

	// XXX there is likely a better pattern for this
	testKeys.Upload, ok = summary["upload"].(float64)
	if !ok {
		err = errors.Wrap(err, "upload key invalid")
	}
	testKeys.Download, ok = summary["download"].(float64)
	if !ok {
		err = errors.Wrap(err, "download key invalid")
	}
	testKeys.Ping, ok = summary["ping"].(float64)
	if !ok {
		err = errors.Wrap(err, "ping key invalid")
	}
	testKeys.MaxRTT, ok = summary["max_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "max_rtt key invalid")
	}
	testKeys.AvgRTT, ok = summary["avg_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "avg_rtt key invalid")
	}
	testKeys.MinRTT, ok = summary["min_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "min_rtt key invalid")
	}
	testKeys.MSS, ok = summary["mss"].(float64)
	if !ok {
		err = errors.Wrap(err, "mss key invalid")
	}
	testKeys.RetransmitRate, ok = summary["retransmit_rate"].(float64)
	if !ok {
		err = errors.Wrap(err, "retransmit_rate key invalid")
	}
	return testKeys, err
}

// LogSummary writes the summary to the standard output
func (n NDT) LogSummary(s string) error {
	return nil
}
