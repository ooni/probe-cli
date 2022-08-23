package registry

//
// Registers the `tlsmiddlebox' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlsmiddlebox"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["tlsmiddlebox"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return tlsmiddlebox.NewExperimentMeasurer(
				*config.(*tlsmiddlebox.Config),
			)
		},
		config:      &tlsmiddlebox.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
