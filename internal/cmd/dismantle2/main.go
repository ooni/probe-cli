package main

import (
	"context"
	"path/filepath"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/session"
)

const softwareName = "dismantle"

const softwareVersion = "0.1.0-dev"

func main() {
	log.SetLevel(log.DebugLevel)

	client := session.NewClient(log.Log)
	ctx := context.Background()

	bootstrapRequest := &session.BootstrapRequest{
		SnowflakeRendezvousMethod: "",
		StateDir:                  filepath.Join("testdata", "engine"),
		ProxyURL:                  "psiphon:///",
		SoftwareName:              softwareName,
		SoftwareVersion:           softwareVersion,
		TorArgs:                   nil,
		TorBinary:                 "",
		TempDir:                   filepath.Join("testdata", "tmp"),
		TunnelDir:                 filepath.Join("testdata", "tunnel"),
		VerboseLogging:            false,
	}
	runtimex.Try0(client.Bootstrap(ctx, bootstrapRequest))

	geolocateRequest := &session.GeolocateRequest{}
	location := runtimex.Try1(client.Geolocate(ctx, geolocateRequest))
	log.Infof("%+v", location)

	checkInRequest := &session.CheckInRequest{
		Charging:        true,
		OnWiFi:          true,
		Platform:        platform.Name(),
		ProbeASN:        location.ProbeASNString(),
		ProbeCC:         location.ProbeCC(),
		RunType:         "manual",
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{},
		},
	}
	checkInResponse := runtimex.Try1(client.CheckIn(ctx, checkInRequest))
	log.Infof("%+v", checkInResponse)

	runtimex.Assert(checkInResponse.Tests.WebConnectivity != nil, "no web connectivity info")

	reportID := checkInResponse.Tests.WebConnectivity.ReportID
	testStartTime := time.Now()
	for _, entry := range checkInResponse.Tests.WebConnectivity.URLs {
		webConnectivityRequest := &session.WebConnectivityRequest{
			Input:         entry.URL,
			ReportID:      reportID,
			TestStartTime: testStartTime,
		}
		measurement, err := client.WebConnectivity(ctx, webConnectivityRequest)
		if err != nil {
			log.Warnf("webconnectivity: measure: %s", err.Error())
			continue
		}
		if err := client.Submit(ctx, measurement); err != nil {
			log.Warnf("webconnectivity: submit: %s", err.Error())
			continue
		}
	}
}
