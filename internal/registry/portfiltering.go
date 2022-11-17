package registry

//
// Registers the 'portfiltering' experiment
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/portfiltering"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["portfiltering"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return portfiltering.NewExperimentMeasurer(
				*config.(*portfiltering.Config),
			)
		},
		config:      &portfiltering.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
