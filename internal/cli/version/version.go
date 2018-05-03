package version

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
)

const Version = "3.0.0-dev.0"

func init() {
	cmd := root.Command("version", "Show version.")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		fmt.Println(Version)
		return nil
	})
}
