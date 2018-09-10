package performance

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
)

// NDT test implementation
type NDT struct {
}

// Run starts the test
func (n NDT) Run(ctl *nettests.Controller) error {
	nt := mk.NewNettest("Ndt")
	ctl.Init(nt)
	return nt.Run()
}

// NDTTestKeys for the test
type NDTTestKeys struct {
	Upload     int64   `json:"upload"`
	Download   int64   `json:"download"`
	Ping       int64   `json:"ping"`
	MaxRTT     float64 `json:"max_rtt"`
	AvgRTT     float64 `json:"avg_rtt"`
	MinRTT     float64 `json:"min_rtt"`
	MSS        int64   `json:"mss"`
	OutOfOrder int64   `json:"out_of_order"`
	PacketLoss float64 `json:"packet_loss"`
	Timeouts   int64   `json:"timeouts"`
	IsAnomaly  bool    `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (n NDT) GetTestKeys(tk map[string]interface{}) interface{} {
	simple := tk["simple"].(map[string]interface{})
	advanced := tk["advanced"].(map[string]interface{})

	return NDTTestKeys{
		Upload:     int64(simple["upload"].(float64)),
		Download:   int64(simple["download"].(float64)),
		Ping:       int64(simple["ping"].(float64)),
		MaxRTT:     advanced["max_rtt"].(float64),
		AvgRTT:     advanced["avg_rtt"].(float64),
		MinRTT:     advanced["min_rtt"].(float64),
		MSS:        int64(advanced["mss"].(float64)),
		OutOfOrder: int64(advanced["out_of_order"].(float64)),
		PacketLoss: advanced["packet_loss"].(float64),
		Timeouts:   int64(advanced["timeouts"].(float64)),
	}
}

// LogSummary writes the summary to the standard output
func (n NDT) LogSummary(s string) error {
	return nil
}
