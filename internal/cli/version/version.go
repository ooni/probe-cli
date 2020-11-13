package version

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/version"
)

func init() {
	cmd := root.Command("version", "Show version.")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		fmt.Println(version.Version)
		return nil
	})
}
