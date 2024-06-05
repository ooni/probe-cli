package registry

//
// Registers the `torsf' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/torsf"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["torsf"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return torsf.NewExperimentMeasurer(
					*config.(*torsf.Config),
				)
			},
			config:           &torsf.Config{},
			enabledByDefault: false,
			inputPolicy:      model.InputNone,
		}
	}
}
