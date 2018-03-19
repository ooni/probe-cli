package performance

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/openobservatory/gooni/nettests"
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

// NDTSummary for the test
type NDTSummary struct {
	Upload     int64
	Download   int64
	Ping       int64
	MaxRTT     int64
	AvgRTT     int64
	MinRTT     int64
	MSS        int64
	OutOfOrder int64
	PacketLoss float64
	Timeouts   int64
}

// Summary generates a summary for a test run
func (n NDT) Summary(tk map[string]interface{}) interface{} {
	simple := tk["simple"].(map[string]interface{})
	advanced := tk["advanced"].(map[string]interface{})

	return NDTSummary{
		Upload:     int64(simple["upload"].(float64)),
		Download:   int64(simple["download"].(float64)),
		Ping:       int64(simple["ping"].(float64)),
		MaxRTT:     int64(advanced["max_rtt"].(float64)),
		AvgRTT:     int64(advanced["avg_rtt"].(float64)),
		MinRTT:     int64(advanced["min_rtt"].(float64)),
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
