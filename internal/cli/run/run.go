package run

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/util"
	"github.com/openobservatory/gooni/nettests/groups"
)

func init() {
	cmd := root.Command("run", "Run a test group or OONI Run link")

	nettestGroup := cmd.Arg("name", "the nettest group to run").Required().String()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		util.Log("Starting %s", *nettestGroup)
		config, ooni, err := root.Init()
		if err != nil {
			log.Errorf("%s", err)
			return err
		}
		log.Infof("%s", config)
		log.Infof("%s", ooni)

		groups.Run(*nettestGroup, ooni)
		return nil
	})
}
