package run

import (
	"errors"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/internal/ooni"
	"github.com/ooni/probe-cli/internal/cli/onboard"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/nettests"
)

func runNettestGroup(tg string, ctx *ooni.Context, network *database.Network) error {
	if ctx.IsTerminated() == true {
		log.Debugf("context is terminated, stopping runNettestGroup early")
		return nil
	}

	sess, err := ctx.NewSession()
	if err != nil {
		log.WithError(err).Error("Failed to create a measurement session")
		return err
	}
	defer sess.Close()

	err = sess.MaybeLookupLocation()
	if err != nil {
		log.WithError(err).Error("Failed to lookup the location of the probe")
		return err
	}
	network, err = database.CreateNetwork(ctx.DB, sess)
	if err != nil {
		log.WithError(err).Error("Failed to create the network row")
		return err
	}
	if err := sess.MaybeLookupBackends(); err != nil {
		log.WithError(err).Warn("Failed to discover OONI backends")
		return err
	}

	group, ok := nettests.NettestGroups[tg]
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

	ctx.ListenForSignals()
	ctx.MaybeListenForStdinClosed()
	for i, nt := range group.Nettests {
		if ctx.IsTerminated() == true {
			log.Debugf("context is terminated, stopping group.Nettests early")
			break
		}
		log.Debugf("Running test %T", nt)
		ctl := nettests.NewController(nt, ctx, result, sess)
		ctl.SetNettestIndex(i, len(group.Nettests))
		if err = nt.Run(ctl); err != nil {
			log.WithError(err).Errorf("Failed to run %s", group.Label)
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

	for name := range nettests.NettestGroups {
		nettestGroupNamesBlue = append(nettestGroupNamesBlue, color.BlueString(name))
	}

	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()

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
	circumventionCmd := cmd.Command("circumvention", "")
	circumventionCmd.Action(func(_ *kingpin.ParseContext) error {
		return runNettestGroup("circumvention", ctx, network)
	})
	allCmd := cmd.Command("all", "").Default()
	allCmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Running %s tests", color.BlueString("all"))
		for tg := range nettests.NettestGroups {
			if err := runNettestGroup(tg, ctx, network); err != nil {
				log.WithError(err).Errorf("failed to run %s", tg)
			}
		}
		return nil
	})
}
