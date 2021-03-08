package run

import (
	"runtime"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/onboard"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/nettests"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")
	noCollector := cmd.Flag("no-collector", "Disable uploading measurements to a collector").Bool()

	var probe *ooni.Probe
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

	functionalRun := func(pred func(name string, gr nettests.Group) bool) error {
		for name, group := range nettests.All {
			if pred(name, group) != true {
				continue
			}
			log.Infof("Running %s tests", color.BlueString(name))
			conf := nettests.RunGroupConfig{GroupName: name, Probe: probe}
			if err := nettests.RunGroup(conf); err != nil {
				log.WithError(err).Errorf("failed to run %s", name)
			}
		}
		return nil
	}

	genRunWithGroupName := func(targetName string) func(*kingpin.ParseContext) error {
		return func(*kingpin.ParseContext) error {
			return functionalRun(func(groupName string, gr nettests.Group) bool {
				return groupName == targetName
			})
		}
	}

	websitesCmd := cmd.Command("websites", "")
	inputFile := websitesCmd.Flag("input-file", "File containing input URLs").Strings()
	input := websitesCmd.Flag("input", "Test the specified URL").Strings()
	websitesCmd.Action(func(_ *kingpin.ParseContext) error {
		log.Infof("Running %s tests", color.BlueString("websites"))
		return nettests.RunGroup(nettests.RunGroupConfig{
			GroupName:  "websites",
			Probe:      probe,
			InputFiles: *inputFile,
			Inputs:     *input,
		})
	})

	easyRuns := []string{"im", "performance", "circumvention", "middlebox"}
	for _, name := range easyRuns {
		cmd.Command(name, "").Action(genRunWithGroupName(name))
	}

	unattendedCmd := cmd.Command("unattended", "")
	unattendedCmd.Action(func(_ *kingpin.ParseContext) error {
		// Until we have enabled the check-in API we're called every
		// hour on darwin and we need to self throttle.
		// TODO(bassosimone): switch to check-in and remove this hack.
		switch runtime.GOOS {
		case "darwin", "windows":
			const veryFew = 10
			probe.Config().Nettests.WebsitesURLLimit = veryFew
		}
		return functionalRun(func(name string, gr nettests.Group) bool {
			return gr.UnattendedOK == true
		})
	})

	allCmd := cmd.Command("all", "").Default()
	allCmd.Action(func(_ *kingpin.ParseContext) error {
		return functionalRun(func(name string, gr nettests.Group) bool {
			return true
		})
	})
}
