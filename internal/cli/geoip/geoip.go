package geoip

import (
	"fmt"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/output"
	"github.com/ooni/probe-cli/utils"
)

func init() {
	cmd := root.Command("geoip", "Perform a geoip lookup")

	shouldUpdate := cmd.Flag("update", "Update the geoip database").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		output.SectionTitle("GeoIP lookup")
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		if err = ctx.MaybeDownloadDataFiles(); err != nil {
			log.WithError(err).Error("failed to download data files")
		}

		geoipPath := utils.GeoIPDir(ctx.Home)
		if *shouldUpdate {
			utils.DownloadGeoIPDatabaseFiles(geoipPath)
		}

		loc, err := utils.GeoIPLookup(geoipPath)
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"type":         "table",
			"asn":          fmt.Sprintf("AS%d", loc.ASN),
			"network_name": loc.NetworkName,
			"country_code": loc.CountryCode,
			"ip":           loc.IP,
		}).Info("Looked up your location")

		return nil
	})
}
