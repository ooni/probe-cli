package registry

//
// Registers the `riseupvpn' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["riseupvpn"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return riseupvpn.NewExperimentMeasurer(
				*config.(*riseupvpn.Config),
			)
		},
		config:      &riseupvpn.Config{},
		inputPolicy: model.InputNone,
	}
}
