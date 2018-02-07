package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("show", "Show a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Show")
		return nil
	})
}
