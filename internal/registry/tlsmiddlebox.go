package registry

//
// Registers the `tlsmiddlebox' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlsmiddlebox"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tlsmiddlebox"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return tlsmiddlebox.NewExperimentMeasurer(
					*config.(*tlsmiddlebox.Config),
				)
			},
			config:           &tlsmiddlebox.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	}
}
