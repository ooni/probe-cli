package run

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/database"
	"github.com/openobservatory/gooni/nettests"
	"github.com/openobservatory/gooni/nettests/groups"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	nettestGroup := cmd.Arg("name", "the nettest group to run").Required().String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Starting %s", *nettestGroup)
		_, ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		group, ok := groups.NettestGroups[*nettestGroup]
		if !ok {
			log.Errorf("No test group named %s", *nettestGroup)
			return errors.New("invalid test group name")
		}
		log.Debugf("Running test group %s", group.Label)

		result, err := database.CreateResult(ctx.DB, database.Result{
			Name:      *nettestGroup,
			StartTime: time.Now().UTC(),
		})
		if err != nil {
			log.Errorf("DB result error: %s", err)
			return err
		}

		for _, nt := range group.Nettests {
			log.Debugf("Running test %T", nt)
			msmtPath := filepath.Join(ctx.TempDir,
				fmt.Sprintf("msmt-%T-%s.jsonl", nt,
					time.Now().UTC().Format(time.RFC3339Nano)))

			ctl := nettests.NewController(nt, ctx, result, msmtPath)
			if err = nt.Run(ctl); err != nil {
				log.WithError(err).Errorf("Failed to run %s", group.Label)
				return err
			}
		}
		if err = result.Finished(ctx.DB, group.Summary); err != nil {
			return err
		}
		return nil
	})
}
