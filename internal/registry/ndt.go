package registry

//
// Registers the `ndt' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/ndt7"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["ndt"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return ndt7.NewExperimentMeasurer(
					*config.(*ndt7.Config),
				)
			},
			config:           &ndt7.Config{},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputNone,
		}
	}
}
