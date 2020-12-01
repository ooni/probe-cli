package run

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/internal/cli/onboard"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/nettests"
	"github.com/ooni/probe-cli/internal/ooni"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	var nettestGroupNamesBlue []string
	var probe *ooni.Probe

	for name := range nettests.NettestGroups {
		nettestGroupNamesBlue = append(nettestGroupNamesBlue, color.BlueString(name))
	}

	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		var err error
		probe, err = root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}

		if err = onboard.MaybeOnboarding(probe); err != nil {
			log.WithError(err).Error("failed to perform onboarding")
			return err
		}

		if *noCollector == true {
			probe.Config().Sharing.UploadResults = false
		}
		return nil
	})

	websitesCmd := cmd.Command("websites", "")
	inputFile := websitesCmd.Flag("input-file", "File containing input URLs").Strings()
	input := websitesCmd.Flag("input", "Test the specified URL").Strings()
	websitesCmd.Action(func(_ *kingpin.ParseContext) error {
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName:  "websites",
			Probe:      probe,
			InputFiles: *inputFile,
			Inputs:     *input,
		})
	})
	imCmd := cmd.Command("im", "")
	imCmd.Action(func(_ *kingpin.ParseContext) error {
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName: "im",
			Probe:     probe,
		})
	})
	performanceCmd := cmd.Command("performance", "")
	performanceCmd.Action(func(_ *kingpin.ParseContext) error {
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName: "performance",
			Probe:     probe,
		})
	})
	middleboxCmd := cmd.Command("middlebox", "")
	middleboxCmd.Action(func(_ *kingpin.ParseContext) error {
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName: "middlebox",
			Probe:     probe,
		})
	})
	circumventionCmd := cmd.Command("circumvention", "")
	circumventionCmd.Action(func(_ *kingpin.ParseContext) error {
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName: "circumvention",
			Probe:     probe,
		})
	})
	allCmd := cmd.Command("all", "").Default()
	allCmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Running %s tests", color.BlueString("all"))
		for tg := range nettests.NettestGroups {
			group := nettests.RunGroupConfig{GroupName: tg, Probe: probe}
			if err := nettests.RunGroup(group); err != nil {
				log.WithError(err).Errorf("failed to run %s", tg)
			}
		}
		return nil
	})
}
