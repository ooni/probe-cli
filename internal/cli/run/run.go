package run

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-cli/nettests/groups"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	var nettestGroupNames []string
	for name := range groups.NettestGroups {
		nettestGroupNames = append(nettestGroupNames, color.BlueString(name))
	}

	nettestGroup := cmd.Arg("name",
		fmt.Sprintf("the nettest group to run. Supported tests are: %s",
			strings.Join(nettestGroupNames, ", "))).Required().String()

	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()
	collectorURL := cmd.Flag("collector-url", "Specify the address of a custom collector").String()
	bouncerURL := cmd.Flag("bouncer-url", "Specify the address of a custom bouncer").String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Starting %s", *nettestGroup)
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
		log.Debugf("Using collector %s", ctx.Config.Advanced.CollectorURL)
		log.Debugf("Using bouncer %s", ctx.Config.Advanced.CollectorURL)

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

		network, err := database.CreateNetwork(ctx.DB, ctx.Location)
		if err != nil {
			log.WithError(err).Error("Failed to create the network row")
			return nil
		}

		result, err := database.CreateResult(ctx.DB, ctx.Home, *nettestGroup, network.ID)
		if err != nil {
			log.Errorf("DB result error: %s", err)
			return err
		}

		for _, nt := range group.Nettests {
			log.Debugf("Running test %T", nt)
			ctl := nettests.NewController(nt, ctx, result)
			if err = nt.Run(ctl); err != nil {
				log.WithError(err).Errorf("Failed to run %s", group.Label)
				return err
			}
		}

		if err = result.Finished(ctx.DB); err != nil {
			return err
		}
		return nil
	})
}
