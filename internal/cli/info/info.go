package info

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("info", "Display information about OONI Probe")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		log.WithFields(log.Fields{
			"path": ctx.Home,
		}).Info("Home")
		log.WithFields(log.Fields{
			"path": ctx.TempDir,
		}).Info("TempDir")

		return nil
	})
}
