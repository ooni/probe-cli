package registry

//
// Registers the `tcpping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tcpping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["tcpping"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return tcpping.NewExperimentMeasurer(
				*config.(*tcpping.Config),
			)
		},
		config:      &tcpping.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
