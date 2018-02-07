package run

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	nettestGroup := cmd.Arg("name", "the nettest group to run").Required().String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Starting %s", nettestGroup)
		return nil
	})
}
