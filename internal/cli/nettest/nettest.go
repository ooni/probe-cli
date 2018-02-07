package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("nettest", "Run a specific nettest")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Nettest")
		return nil
	})
}
