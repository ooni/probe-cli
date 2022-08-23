package registry

//
// Registers the `sniblocking' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/sniblocking"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["sni_blocking"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return sniblocking.NewExperimentMeasurer(
				*config.(*sniblocking.Config),
			)
		},
		config:      &sniblocking.Config{},
		inputPolicy: model.InputOrQueryBackend,
	}
}
