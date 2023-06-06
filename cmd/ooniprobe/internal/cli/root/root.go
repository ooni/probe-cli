package root

import (
	"github.com/alecthomas/kingpin/v2"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/log/handlers/batch"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/log/handlers/cli"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/log/handlers/syslog"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/utils"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Cmd is the root command
var Cmd = kingpin.New("ooniprobe", "")

// Command is syntax sugar for defining sub-commands
var Command = Cmd.Command

// Init should be called by all subcommand that care to have a ooni.Context instance
var Init func() (*ooni.Probe, error)

// NewProbeCLI is like Init but returns a ooni.ProbeCLI instead.
func NewProbeCLI() (ooni.ProbeCLI, error) {
	probeCLI, err := Init()
	if err != nil {
		return nil, err
	}
	return probeCLI, nil
}

func init() {
	configPath := Cmd.Flag("config", "Set a custom config file path").Short('c').String()

	isVerbose := Cmd.Flag("verbose", "Enable verbose log output.").Short('v').Bool()
	isBatch := Cmd.Flag("batch", "Enable batch command line usage.").Bool()
	logHandler := Cmd.Flag(
		"log-handler", "Set the desired log handler (one of: batch, cli, syslog)",
	).String()

	softwareName := Cmd.Flag(
		"software-name", "Override application name",
	).Default(ooni.DefaultSoftwareName).String()
	softwareVersion := Cmd.Flag(
		"software-version", "Override the application version",
	).Default(version.Version).String()
	proxy := Cmd.Flag(
		"proxy", "specify a proxy address for speaking to the OONI Probe backend (use: --proxy=psiphon:/// for psiphon)",
	).String()

	Cmd.PreAction(func(ctx *kingpin.ParseContext) error {
		// TODO(bassosimone): we need to properly deprecate --batch
		// in favour of more granular command line flags.
		if *isBatch && *logHandler != "" {
			log.Fatal("cannot specify --batch and --log-handler together")
		}
		if *isBatch {
			*logHandler = "batch"
		}
		switch *logHandler {
		case "batch":
			log.SetHandler(batch.Default)
		case "cli", "":
			log.SetHandler(cli.Default)
		case "syslog":
			log.SetHandler(syslog.Default)
		default:
			log.Fatalf("unknown --log-handler: %s", *logHandler)
		}
		if *isVerbose {
			log.SetLevel(log.DebugLevel)
			log.Debugf("ooni version %s", version.Version)
		}

		Init = func() (*ooni.Probe, error) {
			var err error

			homePath, err := utils.GetOONIHome()
			if err != nil {
				return nil, err
			}

			probe := ooni.NewProbe(*configPath, homePath)
			err = probe.Init(*softwareName, *softwareVersion, *proxy)
			if err != nil {
				return nil, err
			}
			if *isBatch {
				probe.SetIsBatch(true)
			}

			return probe, nil
		}

		return nil
	})
}
