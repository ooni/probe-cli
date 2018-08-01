package upload

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("upload", "Upload a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Uploading")
		log.Error("this function is not implemented")
		return nil
	})
}
