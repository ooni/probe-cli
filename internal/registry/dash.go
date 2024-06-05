package registry

//
// Registers the `dash' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["dash"] = func() *Factory {
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return dash.NewExperimentMeasurer(
					*config.(*dash.Config),
				)
			},
			config:           &dash.Config{},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputNone,
		}
	}
}
