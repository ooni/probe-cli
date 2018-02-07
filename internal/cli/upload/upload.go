package upload

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
)

func init() {
	cmd := root.Command("upload", "Upload a specific measurement")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Uploading")
		return nil
	})
}
