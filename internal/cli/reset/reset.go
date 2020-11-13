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
			log.WithError(err).Error("failed to init root context")
			return err
		}
		// We need to first the DB otherwise the DB will be rewritten on close when
		// we delete the home directory.
		err = ctx.DB().Close()
		if err != nil {
			log.WithError(err).Error("failed to close the DB")
			return err
		}
		if *force == true {
			os.RemoveAll(ctx.Home())
			log.Infof("Deleted %s", ctx.Home())
		} else {
			log.Infof("Run with --force to delete %s", ctx.Home())
		}

		return nil
	})
}
