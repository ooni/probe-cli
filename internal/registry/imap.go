package registry

//
// Registers the 'imap' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/imap"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["imap"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return imap.NewExperimentMeasurer(
				*config.(*imap.Config),
			)
		},
		config:      &imap.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
