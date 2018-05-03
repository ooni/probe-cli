package info

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("info", "Display information about OONI Probe")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Info")
		return nil
	})
}
