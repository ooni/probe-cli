package registry

//
// Registers the `quicping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "quicping"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return quicping.NewExperimentMeasurer(
					*config.(*quicping.Config),
				)
			},
			canonicalName:    canonicalName,
			config:           &quicping.Config{},
			enabledByDefault: true,
			inputPolicy:      model.InputStrictlyRequired,
		}
	}
}
