package registry

//
// Registers the `quicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["quicping"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return quicping.NewExperimentMeasurer(
				*config.(*quicping.Config),
			)
		},
		config:           &quicping.Config{},
		enabledByDefault: true,
		inputPolicy:      model.InputStrictlyRequired,
	}
}
