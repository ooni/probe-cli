package upload

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
)

func init() {
	cmd := root.Command("upload", "Upload a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("Uploading")
		log.Error("this function is not implemented")
		return nil
	})
}
