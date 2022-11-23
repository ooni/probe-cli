package nettest

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
)

func init() {
	cmd := root.Command("show", "Show a specific measurement")
	msmtID := cmd.Arg("id", "the id of the measurement to show").Required().Int64()
	cmd.Action(func(_ *kingpin.ParseContext) error {
		ctx, err := root.Init()
		if err != nil {
			log.WithError(err).Error("failed to initialize root context")
			return err
		}
		msmt, err := ctx.ReadDB().GetMeasurementJSON(*msmtID)
		if err != nil {
			log.Errorf("error: %v", err)
			return err
		}
		output.MeasurementJSON(msmt)
		return nil
	})
}
