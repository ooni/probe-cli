package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type ExperimentMeasurer struct {
	MockExperimentName    func() string
	MockExperimentVersion func() string
	MockGetSummaryKeys    func(*model.Measurement) (any, error)
	MockRun               func(ctx context.Context, args *model.ExperimentArgs) error
}

var _ model.ExperimentMeasurer = &ExperimentMeasurer{}

// ExperimentName implements model.ExperimentMeasurer.
func (em *ExperimentMeasurer) ExperimentName() string {
	return em.MockExperimentName()
}

// ExperimentVersion implements model.ExperimentMeasurer.
func (em *ExperimentMeasurer) ExperimentVersion() string {
	return em.MockExperimentVersion()
}

// GetSummaryKeys implements model.ExperimentMeasurer.
func (em *ExperimentMeasurer) GetSummaryKeys(meas *model.Measurement) (any, error) {
	return em.MockGetSummaryKeys(meas)
}

// Run implements model.ExperimentMeasurer.
func (em *ExperimentMeasurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	return em.MockRun(ctx, args)
}
