package geoip

import (
	"github.com/alecthomas/kingpin"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cli/root"
	"github.com/ooni/probe-cli/v3/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/output"
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

	engine, err := probeCLI.NewProbeEngine()
	if err != nil {
		return err
	}
	defer engine.Close()

	err = engine.MaybeLookupLocation()
	if err != nil {
		return err
	}

	config.Logger.WithFields(log.Fields{
		"type":         "table",
		"asn":          engine.ProbeASNString(),
		"network_name": engine.ProbeNetworkName(),
		"country_code": engine.ProbeCC(),
		"ip":           engine.ProbeIP(),
	}).Info("Looked up your location")

	return nil
}
