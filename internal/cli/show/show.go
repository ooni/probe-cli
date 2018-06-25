package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("show", "Show a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		_, err := root.Init()
		if err != nil {
			log.WithError(err).Error("failed to initialize root context")
			return err
		}
		log.Error("this function is not implemented")

		return nil
	})
}
