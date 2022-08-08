package registry

//
// Registers the `telegram' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["telegram"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return telegram.NewExperimentMeasurer(
				*config.(*telegram.Config),
			)
		},
		config:      &telegram.Config{},
		inputPolicy: model.InputNone,
	}
}
