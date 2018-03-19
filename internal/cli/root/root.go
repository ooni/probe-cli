package root

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	ooni "github.com/openobservatory/gooni"
	"github.com/openobservatory/gooni/internal/database"
	"github.com/openobservatory/gooni/internal/log/handlers/batch"
	"github.com/openobservatory/gooni/internal/log/handlers/cli"
	"github.com/prometheus/common/version"
)

// Cmd is the root command
var Cmd = kingpin.New("ooni", "")

// Command is syntax sugar for defining sub-commands
var Command = Cmd.Command

// Init should be called by all subcommand that care to have a ooni.OONI instance
var Init func() (*ooni.Config, *ooni.Context, error)

func init() {
	configPath := Cmd.Flag("config", "Set a custom config file path").Short('c').String()

	isVerbose := Cmd.Flag("verbose", "Enable verbose log output.").Short('v').Bool()
	isBatch := Cmd.Flag("batch", "Enable batch command line usage.").Bool()

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

		Init = func() (*ooni.Config, *ooni.Context, error) {
			var config *ooni.Config
			var err error

			if *configPath != "" {
				log.Debugf("Reading config file from %s", *configPath)
				config, err = ooni.ReadConfig(*configPath)
			} else {
				log.Debug("Reading default config file")
				config, err = ooni.ReadDefaultConfigPaths()
			}
			if err != nil {
				return nil, nil, err
			}

			dbPath, err := database.DefaultDatabasePath()
			if err != nil {
				return nil, nil, err
			}

			log.Debugf("Connecting to database sqlite3://%s", dbPath)
			db, err := database.Connect(dbPath)
			if err != nil {
				return nil, nil, err
			}

			ctx := ooni.New(config, db)
			err = ctx.Init()
			if err != nil {
				return nil, nil, err
			}

			return config, ctx, nil
		}

		return nil
	})
}
