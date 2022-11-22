package registry

//
// Registers the `dnsping' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/bittorrent"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["bittorrent"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return bittorrent.NewExperimentMeasurer(
				*config.(*bittorrent.Config),
			)
		},
		config:      &bittorrent.Config{},
		inputPolicy: model.InputOrStaticDefault,
	}
}
