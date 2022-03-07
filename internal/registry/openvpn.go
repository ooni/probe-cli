package registry

//
// Registers the `openvpn` experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	allexperiments["openvpn"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return openvpn.NewExperimentMeasurer(
				*config.(*openvpn.Config),
			)
		},
		config:      &openvpn.Config{},
		inputPolicy: model.InputStrictlyRequired,
	}
}
