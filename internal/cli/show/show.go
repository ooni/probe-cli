package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("show", "Show a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Show")
		return nil
	})
}
