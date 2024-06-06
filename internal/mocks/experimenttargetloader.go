package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ExperimentTargetLoader mocks model.ExperimentTargetLoader
type ExperimentTargetLoader struct {
	MockLoad func(ctx context.Context) ([]model.ExperimentTarget, error)
}

var _ model.ExperimentTargetLoader = &ExperimentTargetLoader{}

// Load calls MockLoad
func (eil *ExperimentTargetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	return eil.MockLoad(ctx)
}
