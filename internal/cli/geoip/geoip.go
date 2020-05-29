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

		sess, err := ctx.NewSession()
		if err != nil {
			return err
		}
		defer sess.Close()

		err = sess.MaybeLookupLocation()
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"type":         "table",
			"asn":          sess.ProbeASNString(),
			"network_name": sess.ProbeNetworkName(),
			"country_code": sess.ProbeCC(),
			"ip":           sess.ProbeIP(),
		}).Info("Looked up your location")

		return nil
	})
}
