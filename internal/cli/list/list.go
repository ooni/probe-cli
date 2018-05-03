package list

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("list", "List measurements")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Listing")
		return nil
	})
}
