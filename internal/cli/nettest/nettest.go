package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("nettest", "Run a specific nettest")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Nettest")
		log.Error("this function is not implemented")
		return nil
	})
}
