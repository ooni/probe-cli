package registry

//
// Registers the `riseupvpn' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["riseupvpn"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return riseupvpn.NewExperimentMeasurer(
					*config.(*riseupvpn.Config),
				)
			},
			buildRicherInputExperiment: riseupvpn.NewRicherInputExperiment,
			config:                     &riseupvpn.Config{},
			inputPolicy:                model.InputNone,
		}
	}
}
