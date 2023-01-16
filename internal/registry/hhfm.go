package registry

//
// Registers the `hhfm' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/hhfm"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["http_header_field_manipulation"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return hhfm.NewExperimentMeasurer(
				*config.(*hhfm.Config),
			)
		},
		config:      &hhfm.Config{},
		inputPolicy: model.InputNone,
	}
}
