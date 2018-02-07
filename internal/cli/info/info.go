package info

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("info", "Display information about OONI Probe")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Info")
		return nil
	})
}
