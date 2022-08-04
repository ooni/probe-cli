package main

//
// Meta info tasks
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine"
)

// newMetaInfoExperimentRunner creates a new instance of metaInfoExperimentRunner.
func newMetaInfoExperimentRunner() taskRunner {
	return &metaInfoExperimentRunner{}
}

// metaInfoExperiment is the meta-info-experiment taskRunner.
type metaInfoExperimentRunner struct{}

var _ taskRunner = &metaInfoExperimentRunner{}

// main implements taskRunner.main.
func (r *metaInfoExperimentRunner) main(ctx context.Context, emitter taskMaybeEmitter, args []byte) {
	for _, exp := range engine.AllExperimentsInfo() {
		ev := &MetaInfoExperimentEventValue{
			Name:      exp.Name,
			UsesInput: exp.InputPolicy != engine.InputNone,
		}
		emitter.maybeEmitEvent(MetaInfoExperimentEventName, ev)
	}
}
