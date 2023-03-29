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
		BackendURL:                "",
		ProxyURL:                  "",
		SnowflakeRendezvousMethod: "",
		SoftwareName:              "miniooni",
		SoftwareVersion:           "0.1.0-dev",
		StateDir:                  filepath.Join("x", "state"),
		TempDir:                   filepath.Join("x", "tmp"),
		TorArgs:                   []string{},
		TorBinary:                 "",
		TunnelDir:                 filepath.Join("x", "tunnel"),
		Verbose:                   false,
	}

	// create session
	sess := runtimex.Try1(miniengine.NewSession(sessionConfig))
	defer sess.Close()

	// bootstrap the session
	bootstrapTask := sess.Bootstrap(ctx)
	awaitTask(bootstrapTask)
	_ = runtimex.Try1(bootstrapTask.Result())

	// geolocate the probe
	locationTask := sess.Geolocate(ctx)
	awaitTask(locationTask)
	location := runtimex.Try1(locationTask.Result())
	log.Infof("%+v", location)

	// call the check-in API
	checkInConfig := &model.OOAPICheckInConfig{
		Charging:        false,
		OnWiFi:          false,
		Platform:        sess.Platform(),
		ProbeASN:        location.ProbeASNString,
		ProbeCC:         location.ProbeCC,
		RunType:         model.RunTypeTimed,
		SoftwareName:    sess.SoftwareName(),
		SoftwareVersion: sess.SoftwareVersion(),
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{},
		},
	}
	checkInTask := sess.CheckIn(ctx, checkInConfig)
	awaitTask(checkInTask)
	checkInResult := runtimex.Try1(checkInTask.Result())
	log.Infof("%+v", checkInResult)

	// create an instance of the Web Connectivity experiment
	exp := runtimex.Try1(sess.NewExperiment("web_connectivity", make(map[string]any)))

	// measure and submit all the URLs
	runtimex.Assert(checkInResult.Tests.WebConnectivity != nil, "nil WebConnectivity")
	for _, entry := range checkInResult.Tests.WebConnectivity.URLs {
		// perform the measurement
		measurementTask := sess.Measure(ctx, exp, entry.URL)
		awaitTask(measurementTask)
		measurementResult := runtimex.Try1(measurementTask.Result())
		log.Infof("%+v", measurementResult)

		// submit the measurement
		submitTask := sess.Submit(ctx, measurementResult.Measurement)
		awaitTask(submitTask)
		reportID := runtimex.Try1(submitTask.Result())
		log.Infof("%+v", reportID)
	}
}
