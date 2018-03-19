package run

import (
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
		group := groups.NettestGroups[*nettestGroup]
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
				fmt.Sprintf("msmt-%s-%T.jsonl", nt,
					time.Now().UTC().Format(time.RFC3339Nano)))

			ctl := nettests.NewController(nt, ctx, result, msmtPath)
			if err := nt.Run(ctl); err != nil {
				log.WithError(err).Errorf("Failed to run %s", group.Label)
				return err
			}
			// XXX
			// 1. Generate the summary
			// 2. Link the measurement to the Result (this should probably happen in
			// the nettest class)
			// 3. Update the summary of the result and the other metadata in the db
			// 4. Move the msmtPath into the final location ~/.ooni/msmts/
		}
		// result.Update(ctx.DB)
		return nil
	})
}
