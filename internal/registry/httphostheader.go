package registry

//
// Registers the `httphostheader' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/httphostheader"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["http_host_header"] = &Factory{
		buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
			return httphostheader.NewExperimentMeasurer(
				*config.(*httphostheader.Config),
			)
		},
		buildRicherInputExperiment: httphostheader.NewRicherInputExperiment,
		config:                     &httphostheader.Config{},
		enabledByDefault:           true,
		inputPolicy:                model.InputOrQueryBackend,
	}
}
