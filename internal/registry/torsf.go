package registry

//
// Registers the `torsf' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/torsf"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["torsf"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return torsf.NewExperimentMeasurer(
				*config.(*torsf.Config),
			)
		},
		config:      &torsf.Config{},
		inputPolicy: model.InputNone,
	}
}
