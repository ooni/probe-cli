package onboard

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/ooni/probe-cli/v3/internal/cli/root"
	"github.com/ooni/probe-cli/v3/internal/config"
	"github.com/ooni/probe-cli/v3/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/output"
	"github.com/pkg/errors"
	"gopkg.in/AlecAivazis/survey.v1"
)

// Onboarding start the interactive onboarding procedure
func Onboarding(config *config.Config) error {
	output.SectionTitle("What is OONI Probe?")

	fmt.Println()
	output.Paragraph("Your tool for detecting internet censorship!")
	fmt.Println()
	output.Paragraph("OONI Probe checks whether your provider blocks access to sites and services. Run OONI Probe to collect evidence of internet censorship and to measure your network performance.")
	fmt.Println()
	err := output.PressEnterToContinue("Press 'Enter' to continue...")
	if err != nil {
		return err
	}

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
	err = output.PressEnterToContinue("Press 'Enter' to continue...")
	if err != nil {
		return err
	}

	output.SectionTitle("Pop Quiz!")
	output.Paragraph("")
	answer := ""
	quiz1 := &survey.Select{
		Message: "Anyone monitoring my internet activity may be able to see that I am running OONI Probe.",
		Options: []string{"true", "false"},
		Default: "true",
	}
	if err := survey.AskOne(quiz1, &answer, nil); err != nil {
		return err
	}
	if answer != "true" {
		output.Paragraph(color.RedString("Actually..."))
		output.Paragraph("OONI Probe is not a privacy tool. Therefore, anyone monitoring your internet activity may be able to see which software you are running.")
	} else {
		output.Paragraph(color.BlueString("Good job!"))
	}
	answer = ""
	quiz2 := &survey.Select{
		Message: "The network data I will collect will automatically be published (unless I opt-out in the settings).",
		Options: []string{"true", "false"},
		Default: "true",
	}
	if err := survey.AskOne(quiz2, &answer, nil); err != nil {
		return err
	}
	if answer != "true" {
		output.Paragraph(color.RedString("Actually..."))
		output.Paragraph("The network data you will collect will automatically be published to increase transparency of internet censorship (unless you opt-out in the settings).")
	} else {
		output.Paragraph(color.BlueString("Well done!"))
	}

	changeDefaults := false
	prompt := &survey.Confirm{
		Message: "Do you want to change the default settings?",
		Default: false,
	}
	if err := survey.AskOne(prompt, &changeDefaults, nil); err != nil {
		return err
	}

	settings := struct {
		IncludeIP        bool
		IncludeNetwork   bool
		UploadResults    bool
		SendCrashReports bool
	}{}
	settings.IncludeIP = false
	settings.IncludeNetwork = true
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

		if err := survey.Ask(qs, &settings); err != nil {
			log.WithError(err).Error("there was an error in parsing your responses")
			return err
		}
	}

	config.Lock()
	config.InformedConsent = true
	config.Advanced.SendCrashReports = settings.SendCrashReports
	config.Sharing.UploadResults = settings.UploadResults
	config.Unlock()

	if err := config.Write(); err != nil {
		log.WithError(err).Error("failed to write config file")
		return err
	}
	return nil
}

// MaybeOnboarding will run the onboarding process only if the informed consent
// config option is set to false
func MaybeOnboarding(probe *ooni.Probe) error {
	if probe.Config().InformedConsent == false {
		if probe.IsBatch() == true {
			return errors.New("cannot run onboarding in batch mode")
		}
		if err := Onboarding(probe.Config()); err != nil {
			return errors.Wrap(err, "onboarding")
		}
	}
	return nil
}

func init() {
	cmd := root.Command("onboard", "Starts the onboarding process")

	yes := cmd.Flag("yes", "Answer yes to all the onboarding questions.").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		probe, err := root.Init()
		if err != nil {
			return err
		}

		if *yes == true {
			probe.Config().Lock()
			probe.Config().InformedConsent = true
			probe.Config().Unlock()

			if err := probe.Config().Write(); err != nil {
				log.WithError(err).Error("failed to write config file")
				return err
			}
			return nil
		}
		if probe.IsBatch() == true {
			return errors.New("cannot do onboarding in batch mode")
		}

		return Onboarding(probe.Config())
	})
}
