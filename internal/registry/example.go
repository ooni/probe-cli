package registry

//
// Registers the `example' experiment.
//

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["example"] = func() *Factory {
		return &Factory{
			buildMeasurer: func(config interface{}) model.ExperimentMeasurer {
				return example.NewExperimentMeasurer(
					*config.(*example.Config), "example",
				)
			},
			buildRicherInputExperiment: example.NewRicherInputExperiment,
			config: &example.Config{
				Message:   "Good day from the example experiment!",
				SleepTime: int64(time.Second),
			},
			enabledByDefault: true,
			interruptible:    true,
			inputPolicy:      model.InputNone,
		}
	}
}
