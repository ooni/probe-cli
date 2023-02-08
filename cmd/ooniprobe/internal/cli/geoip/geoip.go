package geoip

import (
	"context"

	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
)

func init() {
	cmd := root.Command("geoip", "Perform a geoip lookup")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		return dogeoip(defaultconfig)
	})
}

type dogeoipconfig struct {
	Logger       log.Interface
	NewProbeCLI  func() (ooni.ProbeCLI, error)
	SectionTitle func(string)
}

var defaultconfig = dogeoipconfig{
	Logger:       log.Log,
	NewProbeCLI:  root.NewProbeCLI,
	SectionTitle: output.SectionTitle,
}

func dogeoip(config dogeoipconfig) error {
	config.SectionTitle("GeoIP lookup")
	probeCLI, err := config.NewProbeCLI()
	if err != nil {
		return err
	}

	sess := probeCLI.NewSession(context.Background(), "manual")
	defer sess.Close()

	if err := sess.Bootstrap(context.Background()); err != nil {
		log.WithError(err).Error("Failed to bootstrap the measurement session")
		return err
	}

	location, err := sess.Geolocate(context.Background())
	if err != nil {
		log.WithError(err).Error("Failed to lookup the location of the probe")
		return err
	}

	config.Logger.WithFields(log.Fields{
		"type":         "table",
		"asn":          location.ProbeASNString(),
		"network_name": location.ProbeNetworkName(),
		"country_code": location.ProbeCC(),
		"ip":           location.ProbeIP(),
	}).Info("Looked up your location")

	return nil
}
