package registry

//
// Registers the 'smtp' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/smtp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["smtp"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return smtp.NewExperimentMeasurer(
				*config.(*smtp.Config),
			)
		},
		config:      &smtp.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
