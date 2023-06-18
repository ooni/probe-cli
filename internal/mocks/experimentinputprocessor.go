package mocks

import "context"

// ExperimentInputProcessor processes inputs running the given experiment.
type ExperimentInputProcessor struct {
	MockRun func(ctx context.Context) error
}

func (eip *ExperimentInputProcessor) Run(ctx context.Context) error {
	return eip.MockRun(ctx)
}
