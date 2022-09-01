package tlsmiddlebox

//
// Summary
//

import "github.com/ooni/probe-cli/v3/internal/model"

// Summary contains the summary results
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	// TODO(DecFox): Add anomaly logic to generate summary keys for the experiment
	return SummaryKeys{IsAnomaly: false}, nil
}
