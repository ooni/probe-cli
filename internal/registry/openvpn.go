package registry

//
// Registers the `openvpn' experiment.
//

import (
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	AllExperiments["openvpn"] = &Factory{
		build: func(config interface{}) model.ExperimentMeasurer {
			return openvpn.NewExperimentMeasurer(
				*config.(*openvpn.Config), "openvpn",
			)
		},
		config:           &openvpn.Config{},
		enabledByDefault: false,
		interruptible:    true,
		inputPolicy:      model.InputOptional,
	}
}
