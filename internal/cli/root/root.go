package root

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/log/handlers/batch"
	"github.com/ooni/probe-cli/internal/log/handlers/cli"
	"github.com/ooni/probe-cli/internal/utils"
	"github.com/ooni/probe-cli/internal/version"
)

// Cmd is the root command
var Cmd = kingpin.New("ooniprobe", "")

// Command is syntax sugar for defining sub-commands
var Command = Cmd.Command

// Init should be called by all subcommand that care to have a ooni.Context instance
var Init func() (*ooni.Context, error)

func init() {
	configPath := Cmd.Flag("config", "Set a custom config file path").Short('c').String()

	isVerbose := Cmd.Flag("verbose", "Enable verbose log output.").Short('v').Bool()
	isBatch := Cmd.Flag("batch", "Enable batch command line usage.").Bool()

	softwareName := Cmd.Flag(
		"software-name", "Override application name",
	).Default("ooniprobe-cli").String()
	softwareVersion := Cmd.Flag(
		"software-version", "Override the application version",
	).Default(version.Version).String()

	Cmd.PreAction(func(ctx *kingpin.ParseContext) error {
		if *isBatch {
			log.SetHandler(batch.Default)
		} else {
			log.SetHandler(cli.Default)
		}
		if *isVerbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("ooni version %s", version.Version)
		}

		Init = func() (*ooni.Context, error) {
			var err error

			homePath, err := utils.GetOONIHome()
			if err != nil {
				return nil, err
			}

			ctx := ooni.NewContext(*configPath, homePath)
			err = ctx.Init(*softwareName, *softwareVersion)
			if err != nil {
				return nil, err
			}
			if *isBatch {
				ctx.IsBatch = true
			}

			return ctx, nil
		}

		return nil
	})
}
