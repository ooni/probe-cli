// Package run contains code to run other experiments.
//
// This code is currently alpha.
package run

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Config contains settings.
type Config struct{}

// Measurer runs the measurement.
type Measurer struct{}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (Measurer) ExperimentName() string {
	return "run"
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (Measurer) ExperimentVersion() string {
	return "0.2.0"
}

// StructuredInput contains structured input for this experiment.
type StructuredInput struct {
	// Annotations contains extra annotations to add to the
	// final measurement.
	Annotations map[string]string `json:"annotations"`

	// DNSCheck contains settings for the dnscheck experiment.
	DNSCheck dnscheck.Config `json:"dnscheck"`

	// URLGetter contains settings for the urlgetter experiment.
	URLGetter urlgetter.Config `json:"urlgetter"`

	// Name is the name of the experiment to run.
	Name string `json:"name"`

	// Input is the input for this experiment.
	Input string `json:"input"`
}

// Run implements ExperimentMeasurer.ExperimentVersion.
func (Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	var input StructuredInput
	if err := json.Unmarshal([]byte(measurement.Input), &input); err != nil {
		return err
	}
	exprun, found := table[input.Name]
	if !found {
		return fmt.Errorf("no such experiment: %s", input.Name)
	}
	measurement.AddAnnotations(input.Annotations)
	return exprun.do(ctx, input, sess, measurement, callbacks)
}

// GetSummaryKeys implements ExperimentMeasurer.GetSummaryKeys
func (Measurer) GetSummaryKeys(*model.Measurement) (interface{}, error) {
	// TODO(bassosimone): we could extend this interface to call the
	// specific GetSummaryKeys of the experiment we're running.
	return dnscheck.SummaryKeys{IsAnomaly: false}, nil
}

// NewExperimentMeasurer creates a new model.ExperimentMeasurer
// implementing the run experiment.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{}
}
