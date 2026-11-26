package registry

//
// Registers the `torsf' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/torsf"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "torsf"
	register(canonicalName, "v0.5", func() *Factory {
		return &Factory{
			build: func(config any) model.ExperimentMeasurer {
				return torsf.NewExperimentMeasurer(
					*config.(*torsf.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &torsf.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputNone,
		}
	})
}
