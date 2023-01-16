package registry

//
// Registers the `signal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/signal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["signal"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return signal.NewExperimentMeasurer(
				*config.(*signal.Config),
			)
		},
		config:      &signal.Config{},
		inputPolicy: model.InputNone,
	}
}
