package run

import (
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-cli/nettests/groups"
	"github.com/ooni/probe-cli/utils"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	nettestGroup := cmd.Arg("name", "the nettest group to run").Required().String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Starting %s", *nettestGroup)
		ctx, err := root.Init()
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

		err = ctx.MaybeLocationLookup()
		if err != nil {
			log.WithError(err).Error("Failed to lookup the location of the probe")
			return err
		}

		result, err := database.CreateResult(ctx.DB, ctx.Home, database.Result{
			Name:        *nettestGroup,
			StartTime:   time.Now().UTC(),
			Country:     ctx.Location.CountryCode,
			NetworkName: ctx.Location.NetworkName,
			ASN:         fmt.Sprintf("%d", ctx.Location.ASN),
		})
		if err != nil {
			log.Errorf("DB result error: %s", err)
			return err
		}

		for _, nt := range group.Nettests {
			log.Debugf("Running test %T", nt)
			msmtPath := filepath.Join(ctx.TempDir,
				fmt.Sprintf("msmt-%T-%s.jsonl", nt,
					time.Now().UTC().Format(utils.ResultTimestamp)))

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
