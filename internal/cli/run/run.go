package run

import (
	"errors"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/cli/onboard"
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
	var ctx *ooni.Context
	var network *database.Network

	for name := range groups.NettestGroups {
		nettestGroupNamesBlue = append(nettestGroupNamesBlue, color.BlueString(name))
	}

	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()
	collectorURL := cmd.Flag("collector-url", "Specify the address of a custom collector").String()
	bouncerURL := cmd.Flag("bouncer-url", "Specify the address of a custom bouncer").String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		var err error
		ctx, err = root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		if err = onboard.MaybeOnboarding(ctx); err != nil {
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
		network, err = database.CreateNetwork(ctx.DB, ctx.Session)
		if err != nil {
			log.WithError(err).Error("Failed to create the network row")
			return err
		}
		if ctx.Config.Advanced.BouncerURL != "" {
			ctx.Session.AddAvailableHTTPSBouncer(ctx.Config.Advanced.BouncerURL)
		}
		if ctx.Config.Sharing.UploadResults && ctx.Config.Advanced.CollectorURL != "" {
			ctx.Session.AddAvailableHTTPSCollector(ctx.Config.Advanced.CollectorURL)
		}
		if err := ctx.Session.MaybeLookupBackends(); err != nil {
			log.WithError(err).Warn("Failed to discover OONI backends")
			return err
		}
		// Make sure we share what the user wants us to share.
		ctx.Session.SetIncludeProbeIP(ctx.Config.Sharing.IncludeIP)
		ctx.Session.SetIncludeProbeASN(ctx.Config.Sharing.IncludeASN)
		ctx.Session.SetIncludeProbeCC(ctx.Config.Sharing.IncludeCountry)
		return nil
	})

	websitesCmd := cmd.Command("websites", "")
	websitesCmd.Action(func(_ *kingpin.ParseContext) error {
		return runNettestGroup("websites", ctx, network)
	})
	imCmd := cmd.Command("im", "")
	imCmd.Action(func(_ *kingpin.ParseContext) error {
		return runNettestGroup("im", ctx, network)
	})
	performanceCmd := cmd.Command("performance", "")
	performanceCmd.Action(func(_ *kingpin.ParseContext) error {
		return runNettestGroup("performance", ctx, network)
	})
	middleboxCmd := cmd.Command("middlebox", "")
	middleboxCmd.Action(func(_ *kingpin.ParseContext) error {
		return runNettestGroup("middlebox", ctx, network)
	})
	allCmd := cmd.Command("all", "").Default()
	allCmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Running %s tests", color.BlueString("all"))
		for tg := range groups.NettestGroups {
			if err := runNettestGroup(tg, ctx, network); err != nil {
				log.WithError(err).Errorf("failed to run %s", tg)
			}
		}
		return nil
	})
}
