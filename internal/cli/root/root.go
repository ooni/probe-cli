package root

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	ooni "github.com/openobservatory/gooni"
	"github.com/prometheus/common/version"
)

// Cmd is the root command
var Cmd = kingpin.New("ooni", "")

// Command is syntax sugar for defining sub-commands
var Command = Cmd.Command

// Init should be called by all subcommand that care to have a ooni.OONI instance
var Init func() (*ooni.Config, *ooni.OONI, error)

func init() {
	configPath := Cmd.Flag("config", "Set a custom config file path").Short('c').String()
	verbose := Cmd.Flag("verbose", "Enable verbose log output.").Short('v').Bool()

	Cmd.PreAction(func(ctx *kingpin.ParseContext) error {
		log.SetHandler(cli.Default)
		if *verbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("ooni version %s", version.Version)
		}

		Init = func() (*ooni.Config, *ooni.OONI, error) {
			var c *ooni.Config
			var err error

			if *configPath != "" {
				c, err = ooni.ReadConfig(*configPath)
			} else {
				c, err = ooni.ReadDefaultConfigPaths()
			}
			if err != nil {
				return nil, nil, err
			}
			o := ooni.New(c)
			o.Init()
			return c, o, nil
		}

		return nil
	})
}
