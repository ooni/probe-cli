package onboard

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/onboard"
)

func init() {
	cmd := root.Command("onboard", "Starts the onboarding process")

	yes := cmd.Flag("yes", "Answer yes to all the onboarding questions.").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		if *yes == true {
			ctx.Config.Lock()
			ctx.Config.InformedConsent = true
			ctx.Config.Unlock()

			if err := ctx.Config.Write(); err != nil {
				log.WithError(err).Error("failed to write config file")
				return err
			}
			return nil
		}

		return onboard.Onboarding(ctx.Config)
	})
}
