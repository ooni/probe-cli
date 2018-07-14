package version

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/cli/root"
)


func init() {
	cmd := root.Command("version", "Show version.")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		fmt.Println(ooni.Version)
		return nil
	})
}
