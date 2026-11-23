package registry

//
// Registers the `sniblocking' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/sniblocking"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "sni_blocking"
	register(canonicalName, "v0.3", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return sniblocking.NewExperimentMeasurer()
			},
			canonicalName:    canonicalName,
			config:           &sniblocking.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
			newLoader:        sniblocking.NewLoader,
		}
	})
}
