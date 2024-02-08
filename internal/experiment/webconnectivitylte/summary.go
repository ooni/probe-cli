package webconnectivitylte

import "github.com/ooni/probe-cli/v3/internal/model"

var _ model.MeasurementSummaryKeysProvider = &TestKeys{}

// SummaryKeys contains summary keys for this experiment.
type SummaryKeys struct {
	Accessible bool   `json:"accessible"`
	Blocking   string `json:"blocking"`
	IsAnomaly  bool   `json:"-"`
}

// MeasurementSummaryKeys implements model.MeasurementSummaryKeysProvider.
func (tk *TestKeys) MeasurementSummaryKeys() model.MeasurementSummaryKeys {
	// TODO(https://github.com/ooni/probe/issues/1684): accessible not computed correctly (which
	// is an issue that needs some extra investigation to understand how to fix it).
	sk := &SummaryKeys{}
	switch v := tk.Blocking.(type) {
	case string:
		sk.IsAnomaly = true
		sk.Blocking = v
	default:
		// nothing
	}
	sk.Accessible = tk.Accessible.UnwrapOr(false)
	return sk
}

// Anomaly implements model.MeasurementSummaryKeys.
func (sk *SummaryKeys) Anomaly() bool {
	return sk.IsAnomaly
}
