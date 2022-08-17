package registry

//
// Registers the `urlgetter' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["urlgetter"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return urlgetter.NewExperimentMeasurer(
				*config.(*urlgetter.Config),
			)
		},
		config:      &urlgetter.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
