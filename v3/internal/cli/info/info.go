package info

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cli/root"
	"github.com/ooni/probe-cli/v3/internal/ooni"
)

func init() {
	cmd := root.Command("info", "Display information about OONI Probe")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		return doinfo(defaultconfig)
	})
}

type doinfoconfig struct {
	Logger      log.Interface
	NewProbeCLI func() (ooni.ProbeCLI, error)
}

var defaultconfig = doinfoconfig{
	Logger:      log.Log,
	NewProbeCLI: root.NewProbeCLI,
}

func doinfo(config doinfoconfig) error {
	probeCLI, err := config.NewProbeCLI()
	if err != nil {
		config.Logger.Errorf("%s", err)
		return err
	}
	config.Logger.WithFields(log.Fields{"path": probeCLI.Home()}).Info("Home")
	config.Logger.WithFields(log.Fields{"path": probeCLI.TempDir()}).Info("TempDir")
	return nil
}
