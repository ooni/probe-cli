package periodic

import (
	"errors"
	"runtime"

	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/periodic/darwin"
)

type service interface {
	Start() error
	Stop() error
}

var implementations = map[string]service{
	"darwin": darwin.Manager{},
}

func getsvc() (service, error) {
	svc, ok := implementations[runtime.GOOS]
	if ok == false {
		return nil, errors.New("periodic: not implemented on this system")
	}
	return svc, nil
}

func init() {
	cmd := root.Command("periodic", "Run automatic tests in the background")
	start := cmd.Command("start", "Start running automatic tests in the background")
	stop := cmd.Command("stop", "Stop running automatic tests in the background")
	start.Action(func(_ *kingpin.ParseContext) error {
		svc, err := getsvc()
		if err != nil {
			return err
		}
		return svc.Start()
	})
	stop.Action(func(_ *kingpin.ParseContext) error {
		svc, err := getsvc()
		if err != nil {
			return err
		}
		return svc.Stop()
	})
}
