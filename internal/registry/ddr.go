package registry

//
// Registers the `ddr' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/ddr"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "ddr"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return ddr.NewExperimentMeasurer(
					*config.(*ddr.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &ddr.Config{},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputNone,
		}
	}
}
