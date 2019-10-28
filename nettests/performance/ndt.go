package performance

import (
	"github.com/ooni/probe-cli/nettests"
	"github.com/pkg/errors"
)

// NDT test implementation
type NDT struct {
}

// Run starts the test
func (n NDT) Run(ctl *nettests.Controller) error {
	builder, err := ctl.Ctx.Session.NewExperimentBuilder("ndt")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// NDTTestKeys for the test
type NDTTestKeys struct {
	Upload     float64 `json:"upload"`
	Download   float64 `json:"download"`
	Ping       float64 `json:"ping"`
	MaxRTT     float64 `json:"max_rtt"`
	AvgRTT     float64 `json:"avg_rtt"`
	MinRTT     float64 `json:"min_rtt"`
	MSS        float64 `json:"mss"`
	OutOfOrder float64 `json:"out_of_order"`
	PacketLoss float64 `json:"packet_loss"`
	Timeouts   float64 `json:"timeouts"`
	IsAnomaly  bool    `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (n NDT) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var err error
	testKeys := NDTTestKeys{IsAnomaly: false}

	simple, ok := tk["simple"].(map[string]interface{})
	if !ok {
		return testKeys, errors.New("simple key is invalid")
	}
	advanced, ok := tk["advanced"].(map[string]interface{})
	if !ok {
		return testKeys, errors.New("advanced key is invalid")
	}

	// XXX there is likely a better pattern for this
	testKeys.Upload, ok = simple["upload"].(float64)
	if !ok {
		err = errors.Wrap(err, "upload key invalid")
	}
	testKeys.Download, ok = simple["download"].(float64)
	if !ok {
		err = errors.Wrap(err, "download key invalid")
	}
	testKeys.Ping, ok = simple["ping"].(float64)
	if !ok {
		err = errors.Wrap(err, "ping key invalid")
	}
	testKeys.MaxRTT, ok = advanced["max_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "max_rtt key invalid")
	}
	testKeys.AvgRTT, ok = advanced["avg_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "avg_rtt key invalid")
	}
	testKeys.MinRTT, ok = advanced["min_rtt"].(float64)
	if !ok {
		err = errors.Wrap(err, "min_rtt key invalid")
	}
	testKeys.MSS, ok = advanced["mss"].(float64)
	if !ok {
		err = errors.Wrap(err, "mss key invalid")
	}
	testKeys.OutOfOrder, ok = advanced["out_of_order"].(float64)
	if !ok {
		err = errors.Wrap(err, "out_of_order key invalid")
	}
	testKeys.PacketLoss, ok = advanced["packet_loss"].(float64)
	if !ok {
		err = errors.Wrap(err, "packet_loss key invalid")
	}
	testKeys.Timeouts, ok = advanced["timeouts"].(float64)
	if !ok {
		err = errors.Wrap(err, "timeouts key invalid")
	}
	return testKeys, err
}

// LogSummary writes the summary to the standard output
func (n NDT) LogSummary(s string) error {
	return nil
}
