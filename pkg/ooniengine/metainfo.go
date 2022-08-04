package main

//
// Meta info tasks
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
)

func init() {
	taskRegistry["ExperimentMetaInfo"] = newExperimentMetaInfoRunner()
}

// newExperimentMetaInfoRunner creates a new instance of experimentMetaInfoRunner.
func newExperimentMetaInfoRunner() taskRunner {
	return &experimentMetaInfoRunner{}
}

// experimentMetaInfo is the meta-info-experiment taskRunner.
type experimentMetaInfoRunner struct{}

var _ taskRunner = &experimentMetaInfoRunner{}

// main implements taskRunner.main.
func (r *experimentMetaInfoRunner) main(ctx context.Context, emitter taskMaybeEmitter, args []byte) {
	for _, exp := range engine.AllExperimentsInfo() {
		ev := &abi.ExperimentMetaInfoEvent{
			Name:      exp.Name,
			UsesInput: exp.InputPolicy != engine.InputNone,
		}
		emitter.maybeEmitEvent("ExperimentMetaInfo", ev)
	}
}
