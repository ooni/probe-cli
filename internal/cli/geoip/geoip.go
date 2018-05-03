package geoip

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/utils"
)

func init() {
	cmd := root.Command("geoip", "Perform a geoip lookup")

	shouldUpdate := cmd.Flag("update", "Update the geoip database").Bool()

	cmd.Action(func(_ *kingpin.ParseContext) error {
		log.Info("geoip")
		ctx, err := root.Init()
		if err != nil {
			return err
		}

		geoipPath := utils.GeoIPDir(ctx.Home)
		if *shouldUpdate {
			utils.DownloadGeoIPDatabaseFiles(geoipPath)
			utils.DownloadLegacyGeoIPDatabaseFiles(geoipPath)
		}

		loc, err := utils.GeoIPLookup(geoipPath)
		if err != nil {
			return err
		}

		log.WithFields(log.Fields{
			"asn":          loc.ASN,
			"network_name": loc.NetworkName,
			"country_code": loc.CountryCode,
			"ip":           loc.IP,
		}).Info("Looked up your location")

		return nil
	})
}
