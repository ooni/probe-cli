package onboard

import (
	"github.com/alecthomas/kingpin"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/onboard"
)

func init() {
	cmd := root.Command("onboard", "Starts the onboarding process")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		return onboard.Onboarding(ctx.Config)
	})
}
