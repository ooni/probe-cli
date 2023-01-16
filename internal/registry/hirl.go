package registry

//
// Registers the `hirl' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/hirl"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["http_invalid_request_line"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return hirl.NewExperimentMeasurer(
				*config.(*hirl.Config),
			)
		},
		config:      &hirl.Config{},
		inputPolicy: model.InputNone,
	}
}
