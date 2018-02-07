package version

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/cli/root"
)

const Version = "0.0.1"

func init() {
	cmd := root.Command("version", "Show version.")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		fmt.Println(Version)
		return nil
	})
}
