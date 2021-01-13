package periodic

import (
	"errors"
	"runtime"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/onboard"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/periodic"
)

var errNotImplemented = errors.New("periodic: not implemented on this platform")

func init() {
	cmd := root.Command("periodic", "Run automatic tests in the background")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		probe, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		if err := onboard.MaybeOnboarding(probe); err != nil {
			log.WithError(err).Error("failed to perform onboarding")
			return err
		}
		return nil
	})

	start := cmd.Command("start", "Start running automatic tests in the background")
	start.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		if err := svc.Start(); err != nil {
			return err
		}
		log.Info("hint: use 'ooniprobe periodic log stream' to follow logs")
		return nil
	})

	stop := cmd.Command("stop", "Stop running automatic tests in the background")
	stop.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.Stop()
	})

	logCmd := cmd.Command("log", "Access background runs logs")
	stream := logCmd.Command("stream", "Stream background runs logs")
	stream.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.LogStream()
	})

	show := logCmd.Command("show", "Show background runs logs")
	show.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		return svc.LogShow()
	})

	status := cmd.Command("status", "Shows periodic instance status")
	status.Action(func(_ *kingpin.ParseContext) error {
		svc := periodic.Get(runtime.GOOS)
		if svc == nil {
			return errNotImplemented
		}
		out, err := svc.Status()
		if err != nil {
			return err
		}
		log.Infof("status: %s", out)
		switch out {
		case periodic.StatusRunning:
			log.Info("hint: use 'ooniprobe periodic stop' to stop")
			log.Info("hint: use 'ooniprobe periodic log stream' to follow logs")
		case periodic.StatusScheduled:
			log.Info("hint: use 'ooniprobe periodic stop' to stop")
			log.Info("hint: use 'ooniprobe periodic log show' to see previous logs")
		case periodic.StatusStopped:
			log.Info("hint: use 'ooniprobe periodic start' to start")
		}
		return nil
	})
}
