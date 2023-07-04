package main

//
// Demonstrates using the fundamental OONI Engine API
//

import (
	"context"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/miniengine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// awaitableTask is a [miniengine.Task] that we can await and for which
// we can obtain the interim events while it's running.
type awaitableTask interface {
	Done() <-chan any
	Events() <-chan *miniengine.Event
}

// awaitTask awaits for the task to be done and emits interim events
func awaitTask(task awaitableTask) {
	for {
		select {
		case <-task.Done():
			return
		case ev := <-task.Events():
			switch ev.EventType {
			case miniengine.EventTypeProgress:
				log.Infof("PROGRESS %f %s", ev.Progress, ev.Message)
			case miniengine.EventTypeInfo:
				log.Infof("%s", ev.Message)
			case miniengine.EventTypeWarning:
				log.Warnf("%s", ev.Message)
			case miniengine.EventTypeDebug:
				log.Debugf("%s", ev.Message)
			}
		}
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()

	// create session config
	sessionConfig := &miniengine.SessionConfig{
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		StateDir:        filepath.Join("x", "state"),
		TempDir:         filepath.Join("x", "tmp"),
		TunnelDir:       filepath.Join("x", "tunnel"),
		Verbose:         false,
	}

	// create session
	sess := runtimex.Try1(miniengine.NewSession(sessionConfig))
	defer sess.Close()

	// create the bootstrap config
	bootstrapConfig := &miniengine.BootstrapConfig{
		BackendURL:                "",
		CategoryCodes:             []string{},
		Charging:                  false,
		OnWiFi:                    false,
		ProxyURL:                  "",
		RunType:                   model.RunTypeTimed,
		SnowflakeRendezvousMethod: "",
		TorArgs:                   []string{},
		TorBinary:                 "",
	}

	// bootstrap the session
	bootstrapTask := sess.Bootstrap(ctx, bootstrapConfig)
	awaitTask(bootstrapTask)
	_ = runtimex.Try1(bootstrapTask.Result())

	// obtain the probe geolocation
	location := runtimex.Try1(sess.GeolocateResult())
	log.Infof("%+v", location)

	// obtain the check-in API response
	checkInResult := runtimex.Try1(sess.CheckInResult())
	log.Infof("%+v", checkInResult)

	// obtain check-in information for the web connectivity experiment
	runtimex.Assert(checkInResult.Tests.WebConnectivity != nil, "nil WebConnectivity")
	webConnectivity := checkInResult.Tests.WebConnectivity

	log.Infof("report ID: %s", webConnectivity.ReportID)

	// measure and submit all the URLs
	for _, entry := range webConnectivity.URLs {
		// perform the measurement
		options := make(map[string]any)
		measurementTask := sess.Measure(ctx, "web_connectivity", options, entry.URL)
		awaitTask(measurementTask)
		measurementResult := runtimex.Try1(measurementTask.Result())
		log.Infof("%+v", measurementResult)

		// set the report ID
		measurementResult.Measurement.ReportID = webConnectivity.ReportID

		// submit the measurement
		submitTask := sess.Submit(ctx, measurementResult.Measurement)
		awaitTask(submitTask)
		_ = runtimex.Try1(submitTask.Result())
		log.Infof(
			"https://explorer.ooni.org/measurement/%s?input=%s",
			webConnectivity.ReportID,
			entry.URL,
		)
	}
}
