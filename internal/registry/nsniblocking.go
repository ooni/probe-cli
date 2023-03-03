package registry

//
// Registers the `nsniblocking' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/nsniblocking"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["nsni_blocking"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return nsniblocking.NewExperimentMeasurer(
				*config.(*nsniblocking.Config),
			)
		},
		config:      &nsniblocking.Config{},
		inputPolicy: model.InputOrQueryBackend,
	}
}
