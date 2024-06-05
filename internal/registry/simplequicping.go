package registry

//
// Registers the `simplequicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/simplequicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["simplequicping"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return simplequicping.NewExperimentMeasurer(
					*config.(*simplequicping.Config),
				)
			},
			config:           &simplequicping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	}
}
