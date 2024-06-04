package registry

//
// Registers the `telegram' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/telegram"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["telegram"] = &Factory{
		build: func(config any) model.ExperimentMeasurer {
			return telegram.NewExperimentMeasurer(
				config.(telegram.Config),
			)
		},
		config:           telegram.Config{},
		enabledByDefault: true,
		interruptible:    false,
		inputPolicy:      model.InputNone,
	}
}
