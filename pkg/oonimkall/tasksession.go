package oonimkall

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

//
// This file implements taskSession and derived types.
//

// taskKVStoreFSBuilderEngine creates a new KVStore
// using the ./internal/engine package.
type taskKVStoreFSBuilderEngine struct{}

var _ taskKVStoreFSBuilder = &taskKVStoreFSBuilderEngine{}

func (*taskKVStoreFSBuilderEngine) NewFS(path string) (model.KeyValueStore, error) {
	return kvstore.NewFS(path)
}

// taskSessionBuilderEngine builds a new session
// using the ./internal/engine package.
type taskSessionBuilderEngine struct{}

var _ taskSessionBuilder = &taskSessionBuilderEngine{}

// NewSession implements taskSessionBuilder.NewSession.
func (b *taskSessionBuilderEngine) NewSession(ctx context.Context,
	config engine.SessionConfig) (taskSession, error) {
	sess, err := engine.NewSession(ctx, config)
	if err != nil {
		return nil, err
	}
	return &taskSessionEngine{sess}, nil
}

// taskSessionEngine wraps ./internal/engine's Session.
type taskSessionEngine struct {
	*engine.Session
}

var _ taskSession = &taskSessionEngine{}

// NewExperimentBuilderByName implements
// taskSessionEngine.NewExperimentBuilderByName.
func (sess *taskSessionEngine) NewExperimentBuilderByName(
	name string) (taskExperimentBuilder, error) {
	builder, err := sess.NewExperimentBuilder(name)
	if err != nil {
		return nil, err
	}
	return &taskExperimentBuilderEngine{builder}, err
}

// taskExperimentBuilderEngine wraps ./internal/engine's
// ExperimentBuilder type.
type taskExperimentBuilderEngine struct {
	*engine.ExperimentBuilder
}

var _ taskExperimentBuilder = &taskExperimentBuilderEngine{}

// NewExperimentInstance implements
// taskExperimentBuilder.NewExperimentInstance.
func (b *taskExperimentBuilderEngine) NewExperimentInstance() taskExperiment {
	return &taskExperimentEngine{b.NewExperiment()}
}

// taskExperimentEngine wraps ./internal/engine's Experiment.
type taskExperimentEngine struct {
	*engine.Experiment
}

var _ taskExperiment = &taskExperimentEngine{}
