package onboard

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/colors"
	"github.com/ooni/probe-cli/internal/output"
	"gopkg.in/AlecAivazis/survey.v1"
)

func init() {
	cmd := root.Command("onboard", "Starts the onboarding process")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		output.SectionTitle("What is OONI Probe?")

		fmt.Println()
		output.Paragraph("Your tool for detecting internet censorship!")
		fmt.Println()
		output.Paragraph("OONI Probe checks whether your provider blocks access to sites and services. Run OONI Probe to collect evidence of internet censorship and to measure your network performance.")
		fmt.Println()
		output.PressEnterToContinue("Press 'Enter' to continue...")

		output.SectionTitle("Heads Up")
		fmt.Println()
		output.Bullet("Anyone monitoring your internet activity (such as your government or ISP) may be able to see that you are running OONI Probe.")
		fmt.Println()
		output.Bullet("The network data you will collect will automatically be published (unless you opt-out in the settings).")
		fmt.Println()
		output.Bullet("You may test objectionable sites.")
		fmt.Println()
		output.Bullet("Read the documentation to learn more.")
		fmt.Println()
		output.PressEnterToContinue("Press 'Enter' to continue...")

		output.SectionTitle("Pop Quiz!")
		output.Paragraph("")
		answer := ""
		quiz1 := &survey.Select{
			Message: "Anyone monitoring my internet activity may be able to see that I am running OONI Probe.",
			Options: []string{"true", "false"},
			Default: "true",
		}
		survey.AskOne(quiz1, &answer, nil)
		if answer != "true" {
			output.Paragraph(colors.Red("Actually..."))
			output.Paragraph("OONI Probe is not a privacy tool. Therefore, anyone monitoring your internet activity may be able to see which software you are running.")
		} else {
			output.Paragraph(colors.Blue("Good job!"))
		}
		answer = ""
		quiz2 := &survey.Select{
			Message: "The network data I will collect will automatically be published (unless I opt-out in the settings).",
			Options: []string{"true", "false"},
			Default: "true",
		}
		survey.AskOne(quiz2, &answer, nil)
		if answer != "true" {
			output.Paragraph(colors.Red("Actually..."))
			output.Paragraph("The network data you will collect will automatically be published to increase transparency of internet censorship (unless you opt-out in the settings).")
		} else {
			output.Paragraph(colors.Blue("Well done!"))
		}

		changeDefaults := false
		prompt := &survey.Confirm{
			Message: "Do you want to change the default settings?",
			Default: false,
		}
		survey.AskOne(prompt, &changeDefaults, nil)

		settings := struct {
			IncludeIP        bool
			IncludeNetwork   bool
			IncludeCountry   bool
			UploadResults    bool
			SendCrashReports bool
		}{}
		settings.IncludeIP = false
		settings.IncludeNetwork = true
		settings.IncludeCountry = true
		settings.UploadResults = true
		settings.SendCrashReports = true

		if changeDefaults == true {
			var qs = []*survey.Question{
				{
					Name:   "IncludeIP",
					Prompt: &survey.Confirm{Message: "Should we include your IP?"},
				},
				{
					Name: "IncludeNetwork",
					Prompt: &survey.Confirm{
						Message: "Can we include your network name?",
						Default: true,
					},
				},
				{
					Name: "IncludeCountry",
					Prompt: &survey.Confirm{
						Message: "Can we include your country name?",
						Default: true,
					},
				},
				{
					Name: "UploadResults",
					Prompt: &survey.Confirm{
						Message: "Can we upload your results?",
						Default: true,
					},
				},
				{
					Name: "SendCrashReports",
					Prompt: &survey.Confirm{
						Message: "Can we send crash reports to OONI?",
						Default: true,
					},
				},
			}

			err := survey.Ask(qs, &settings)
			if err != nil {
				log.WithError(err).Error("there was an error in parsing your responses")
				return err
			}
		}

		ctx.Config.Lock()
		ctx.Config.InformedConsent = true
		ctx.Config.Advanced.IncludeCountry = settings.IncludeCountry
		ctx.Config.Sharing.IncludeIP = settings.IncludeIP
		ctx.Config.Sharing.IncludeASN = settings.IncludeNetwork
		ctx.Config.Sharing.UploadResults = settings.UploadResults
		ctx.Config.Sharing.SendCrashReports = settings.SendCrashReports
		ctx.Config.Unlock()

		if err := ctx.Config.Write(); err != nil {
			log.WithError(err).Error("failed to write config file")
			return err
		}

		return nil
	})
}
