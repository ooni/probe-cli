package registry

//
// Registers the `fbmessenger' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/fbmessenger"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["facebook_messenger"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return fbmessenger.NewExperimentMeasurer(
				*config.(*fbmessenger.Config),
			)
		},
		buildRicherInputExperiment: fbmessenger.NewRicherInputExperiment,
		config:                     &fbmessenger.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputNone,
	}
}
