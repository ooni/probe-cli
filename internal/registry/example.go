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
	const canonicalName = "example"
	AllExperiments[canonicalName] = func() *Factory {
		// TODO(bassosimone,DecFox): as pointed out by @ainghazal, this experiment
		// should be the one that people modify to start out new experiments, so it's
		// kind of suboptimal that it has a constructor with explicit experiment
		// name to ease writing some tests that ./pkg/oonimkall needs given that no
		// other experiment ever sets the experiment name externally!
		return &Factory{
			build: func(config interface{}) model.ExperimentMeasurer {
				return example.NewExperimentMeasurer(
					*config.(*example.Config), "example",
				)
			},
			canonicalName: canonicalName,
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
