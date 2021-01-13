package periodic

import (
	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("periodic", "Run automatic tests in the background")
	start := cmd.Command("start", "Start running automatic tests in the background")
	stop := cmd.Command("stop", "Stop running automatic tests in the background")
	start.Action(func(_ *kingpin.ParseContext) error {
		return nil
	})
	stop.Action(func(_ *kingpin.ParseContext) error {
		return nil
	})
}
