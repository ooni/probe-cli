package registry

//
// Registers the `whatsapp' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["whatsapp"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return whatsapp.NewExperimentMeasurer(
				*config.(*whatsapp.Config),
			)
		},
		buildRicherInputExperiment: whatsapp.NewRicherInputExperiment,
		config:                     &whatsapp.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputNone,
	}
}
