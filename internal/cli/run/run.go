package run

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-cli/nettests/groups"
)

func runNettestGroup(tg string, ctx *ooni.Context, network *database.Network) error {
	group, ok := groups.NettestGroups[tg]
	if !ok {
		log.Errorf("No test group named %s", tg)
		return errors.New("invalid test group name")
	}
	log.Debugf("Running test group %s", group.Label)

	result, err := database.CreateResult(ctx.DB, ctx.Home, tg, network.ID)
	if err != nil {
		log.Errorf("DB result error: %s", err)
		return err
	}

	for i, nt := range group.Nettests {
		log.Debugf("Running test %T", nt)
		ctl := nettests.NewController(nt, ctx, result)
		ctl.SetNettestIndex(i, len(group.Nettests))
		if err = nt.Run(ctl); err != nil {
			log.WithError(err).Errorf("Failed to run %s", group.Label)
			return err
		}
	}

	if err = result.Finished(ctx.DB); err != nil {
		return err
	}
	return nil
}

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	var nettestGroupNamesBlue []string
	for name := range groups.NettestGroups {
		nettestGroupNamesBlue = append(nettestGroupNamesBlue, color.BlueString(name))
	}

	nettestGroup := cmd.Arg("name",
		fmt.Sprintf("the nettest group to run. Supported tests are: %s, or nothing to run them all",
			strings.Join(nettestGroupNamesBlue, ", "))).String()

	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()
	collectorURL := cmd.Flag("collector-url", "Specify the address of a custom collector").String()
	bouncerURL := cmd.Flag("bouncer-url", "Specify the address of a custom bouncer").String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		if err = ctx.MaybeOnboarding(); err != nil {
			log.WithError(err).Error("failed to perform onboarding")
			return err
		}

		if *noCollector == true {
			ctx.Config.Sharing.UploadResults = false
		}
		if *collectorURL != "" {
			ctx.Config.Advanced.CollectorURL = *collectorURL
		}
		if *bouncerURL != "" {
			ctx.Config.Advanced.BouncerURL = *bouncerURL
		}
		log.Debugf("Using collector: %s", ctx.Config.Advanced.CollectorURL)
		log.Debugf("Using bouncer: %s", ctx.Config.Advanced.CollectorURL)

		err = ctx.MaybeLocationLookup()
		if err != nil {
			log.WithError(err).Error("Failed to lookup the location of the probe")
			return err
		}
		network, err := database.CreateNetwork(ctx.DB, ctx.Session.Location)
		if err != nil {
			log.WithError(err).Error("Failed to create the network row")
			return err
		}
		if ctx.Config.Advanced.BouncerURL != "" {
			ctx.Session.SetAvailableHTTPSBouncer(ctx.Config.Advanced.BouncerURL)
		}
		if err := ctx.Session.LookupBackends(context.Background()); err != nil {
			log.WithError(err).Error("Failed to discover available backends")
			if ctx.Config.Advanced.CollectorURL == "" ||
				ctx.Config.Sharing.UploadResults == false {
				return err
			}
			// Fallthrough if we have a configured collector; we may miss test
			// helpers and thus we'll possibly fail later, but in some cases we
			// may as well continue and successufully run some nettests.
			//
			// Likewise if we are not uploading results as part of testing.
		}
		if ctx.Config.Advanced.CollectorURL != "" {
			ctx.Session.SetAvailableHTTPSBouncer(ctx.Config.Advanced.CollectorURL)
		}

		if *nettestGroup == "" {
			log.Infof("Running %s tests", color.BlueString("all"))
			for tg := range groups.NettestGroups {
				if err := runNettestGroup(tg, ctx, network); err != nil {
					log.WithError(err).Errorf("failed to run %s", tg)
				}
			}
			return nil
		}
		log.Infof("Running %s", color.BlueString(*nettestGroup))
		return runNettestGroup(*nettestGroup, ctx, network)
	})
}
