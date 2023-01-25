package registry

//
// Registers the `tlsping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlsping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["tlsping"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return tlsping.NewExperimentMeasurer(
				*config.(*tlsping.Config),
			)
		},
		config:      &tlsping.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
