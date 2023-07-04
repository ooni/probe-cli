package geoip

import (
	"context"

	"github.com/alecthomas/kingpin/v2"
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/root"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/ooni"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/output"
	"github.com/ooni/probe-cli/v3/internal/miniengine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func init() {
	cmd := root.Command("geoip", "Perform a geoip lookup")
	cmd.Action(func(_ *kingpin.ParseContext) error {
		return dogeoip(defaultconfig)
	})
}

type dogeoipconfig struct {
	Logger       log.Interface
	NewProbeCLI  func() (*ooni.Probe, error)
	SectionTitle func(string)
}

var defaultconfig = dogeoipconfig{
	Logger:       log.Log,
	NewProbeCLI:  root.Init,
	SectionTitle: output.SectionTitle,
}

func dogeoip(config dogeoipconfig) error {
	config.SectionTitle("GeoIP lookup")
	probeCLI, err := config.NewProbeCLI()
	if err != nil {
		return err
	}

	// create a measurement session
	sessConfig := probeCLI.NewSessionConfig(model.RunTypeManual)
	sess, err := miniengine.NewSession(sessConfig)
	if err != nil {
		log.WithError(err).Error("Failed to create a measurement session")
		return err
	}
	defer sess.Close()

	// XXX: not very lightweight to perform a full bootstrap here

	// bootstrap the measurement session
	bootstrapConfig := &miniengine.BootstrapConfig{
		BackendURL:                "",
		CategoryCodes:             []string{},
		Charging:                  true,
		OnWiFi:                    true,
		ProxyURL:                  probeCLI.ProxyURL(),
		RunType:                   model.RunTypeManual,
		SnowflakeRendezvousMethod: "",
		TorArgs:                   []string{},
		TorBinary:                 "",
	}
	bootstrapTask := sess.Bootstrap(context.Background(), bootstrapConfig)
	// XXX: skipping log messages here
	<-bootstrapTask.Done()
	if _, err := bootstrapTask.Result(); err != nil {
		log.WithError(err).Error("Failed to bootstrap a measurement session")
		return err
	}

	location, err := sess.GeolocateResult()
	if err != nil {
		return err
	}

	config.Logger.WithFields(log.Fields{
		"type":         "table",
		"asn":          location.ProbeASNString,
		"network_name": location.ProbeNetworkName,
		"country_code": location.ProbeCC,
		"ip":           location.ProbeIP,
	}).Info("Looked up your location")

	return nil
}
