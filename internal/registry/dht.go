package registry

//
// Registers the `dnsping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dht"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["dht"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return dht.NewExperimentMeasurer(
				*config.(*dht.Config),
			)
		},
		config:      &dht.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
