package registry

//
// Registers the `dash' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["dash"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return dash.NewExperimentMeasurer(
				*config.(*dash.Config),
			)
		},
		config:        &dash.Config{},
		interruptible: true,
		inputPolicy:   model.InputNone,
	}
}
