package mocks

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// ExperimentInputLoader mocks model.ExperimentInputLoader
type ExperimentInputLoader struct {
	MockLoad func(ctx context.Context) ([]model.OOAPIURLInfo, error)
}

var _ model.ExperimentInputLoader = &ExperimentInputLoader{}

// Load calls MockLoad
func (eil *ExperimentInputLoader) Load(ctx context.Context) ([]model.OOAPIURLInfo, error) {
	return eil.MockLoad(ctx)
}
