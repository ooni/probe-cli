package periodic

import (
	"errors"
	"runtime"

	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/periodic"
)

var errNotImplemented = errors.New("periodic: not implemented on this platform")

func init() {
	cmd := root.Command("periodic", "Run automatic tests in the background")
	start := cmd.Command("start", "Start running automatic tests in the background")
	stop := cmd.Command("stop", "Stop running automatic tests in the background")
	start.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.Start()
	})
	stop.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.Stop()
	})
	log := cmd.Command("log", "Access background runs logs")
	stream := log.Command("stream", "Stream background runs logs")
	stream.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.LogStream()
	})
	show := log.Command("show", "Show background runs logs")
	show.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.LogShow()
	})
}
