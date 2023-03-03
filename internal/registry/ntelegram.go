package registry

//
// Registers the `ntelegram' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/ntelegram"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["ntelegram"] = &Factory{
		build: func(config any) model.ExperimentMeasurer {
			return ntelegram.NewExperimentMeasurer(
				config.(ntelegram.Config),
			)
		},
		config:        ntelegram.Config{},
		interruptible: false,
		inputPolicy:   model.InputNone,
	}
}
