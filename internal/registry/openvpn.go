package registry

//
// Registers the `openvpn' experiment.
//

import (
	"time"

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
		config: &openvpn.Config{
			Message:   "This is not an experiment yet!",
			SleepTime: int64(time.Second),
		},
		enabledByDefault: true,
		interruptible:    true,
		inputPolicy:      model.InputNone,
	}
}
