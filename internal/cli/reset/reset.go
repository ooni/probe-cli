package reset

import (
	"os"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
)

func init() {
	cmd := root.Command("reset", "Cleanup an old or experimental installation")
	force := cmd.Flag("force", "Force deleting the OONI Home").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		if *force == true {
			os.RemoveAll(ctx.Home)
			log.Infof("Deleted %s", ctx.Home)
		} else {
			log.Infof("Run with --force to delete %s", ctx.Home)
		}

		return nil
	})
}
