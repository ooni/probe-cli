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
		probeCLI, err := root.Init()
		if err != nil {
			return err
		}

		engine, err := probeCLI.NewProbeEngine()
		if err != nil {
			return err
		}
		defer engine.Close()

		err = engine.MaybeLookupLocation()
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"type":         "table",
			"asn":          engine.ProbeASNString(),
			"network_name": engine.ProbeNetworkName(),
			"country_code": engine.ProbeCC(),
			"ip":           engine.ProbeIP(),
		}).Info("Looked up your location")

		return nil
	})
}
