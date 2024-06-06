package registry

//
// Registers the `vanilla_tor' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/vanillator"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	const canonicalName = "vanilla_tor"
	AllExperiments[canonicalName] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return vanillator.NewExperimentMeasurer(
					*config.(*vanillator.Config),
				)
			},
			canonicalName: canonicalName,
			config:        &vanillator.Config{},
			// We discussed this topic with @aanorbel. On Android this experiment crashes
			// frequently because of https://github.com/ooni/probe/issues/2406. So, it seems
			// more cautious to disable it by default and let the check-in API decide.
			enabledByDefault: false,
			inputPolicy:      model.InputNone,
		}
	}
}
