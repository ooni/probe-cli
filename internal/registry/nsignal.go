package registry

//
// Registers the `nsignal' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/nsignal"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["nsignal"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return nsignal.NewExperimentMeasurer(
				*config.(*nsignal.Config),
			)
		},
		config:      &nsignal.Config{},
		inputPolicy: model.InputNone,
	}
}
