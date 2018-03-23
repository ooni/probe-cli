package geoip

import (
	"path/filepath"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/utils"
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

		geoipPath := filepath.Join(ctx.Home, "geoip")

		if *shouldUpdate {
			utils.DownloadGeoIPDatabaseFiles(geoipPath)
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
