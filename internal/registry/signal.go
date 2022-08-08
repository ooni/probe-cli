package registry

//
// Registers the `signal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["signal"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return signal.NewExperimentMeasurer(
				*config.(*signal.Config),
			)
		},
		config:      &signal.Config{},
		inputPolicy: model.InputNone,
	}
}
