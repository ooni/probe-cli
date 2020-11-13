package info

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("info", "Display information about OONI Probe")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		probeCLI, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		log.WithFields(log.Fields{
			"path": probeCLI.Home(),
		}).Info("Home")
		log.WithFields(log.Fields{
			"path": probeCLI.TempDir(),
		}).Info("TempDir")

		return nil
	})
}
