package registry

//
// Registers the `whatsapp' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["whatsapp"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return whatsapp.NewExperimentMeasurer(
				*config.(*whatsapp.Config),
			)
		},
		config:      &whatsapp.Config{},
		inputPolicy: model.InputNone,
	}
}
