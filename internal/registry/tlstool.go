package registry

//
// Registers the `tlstool' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/tlstool"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "tlstool"
	register(canonicalName, "v0.1", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return tlstool.NewExperimentMeasurer(
					*config.(*tlstool.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &tlstool.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputOrQueryBackend,
		}
	})
}
