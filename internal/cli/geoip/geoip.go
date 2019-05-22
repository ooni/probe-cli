package geoip

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/output"
)

func init() {
	cmd := root.Command("geoip", "Perform a geoip lookup")

	cmd.Action(func(_ *kingpin.ParseContext) error {
		output.SectionTitle("GeoIP lookup")
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		err = ctx.MaybeLocationLookup()
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"type":         "table",
			"asn":          ctx.Session.ProbeASNString(),
			"network_name": ctx.Session.ProbeNetworkName(),
			"country_code": ctx.Session.ProbeCC(),
			"ip":           ctx.Session.ProbeIP(),
		}).Info("Looked up your location")

		return nil
	})
}
