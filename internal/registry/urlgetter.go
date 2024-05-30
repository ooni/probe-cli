package registry

//
// Registers the `urlgetter' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["urlgetter"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return urlgetter.NewExperimentMeasurer(
				*config.(*urlgetter.Config),
			)
		},
		buildRicherInputExperiment: urlgetter.NewRicherInputExperiment,
		config:                     &urlgetter.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputStrictlyRequired,
	}
}
