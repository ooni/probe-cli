package list

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("list", "List measurements")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Listing")
		return nil
	})
}
