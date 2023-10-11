package registry

//
// Registers the `tlstool' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlstool"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tlstool"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return tlstool.NewExperimentMeasurer(
				*config.(*tlstool.Config),
			)
		},
		config:           &tlstool.Config{},
		enabledByDefault: true,
		inputPolicy:      model.InputOrQueryBackend,
	}
}
