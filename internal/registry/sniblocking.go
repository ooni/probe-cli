package registry

//
// Registers the `sniblocking' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/sniblocking"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["sni_blocking"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return sniblocking.NewExperimentMeasurer(
					*config.(*sniblocking.Config),
				)
			},
			config:           &sniblocking.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		}
	}
}
