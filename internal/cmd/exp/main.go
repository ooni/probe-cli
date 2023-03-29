package main

import (
	"context"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/miniengine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/platform"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type awaitableTask interface {
	Done() <-chan any
	Events() <-chan *miniengine.Event
}

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

	// TODO(bassosimone): we want a factory to construct a new OOAPICheckInConfig
	// using the information available to the session.

	// call the check-in API
	checkInConfig := &model.OOAPICheckInConfig{
		Charging:        false,
		OnWiFi:          false,
		Platform:        platform.Name(),
		ProbeASN:        location.ProbeASNString,
		ProbeCC:         location.ProbeCC,
		RunType:         model.RunTypeTimed,
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		WebConnectivity: model.OOAPICheckInConfigWebConnectivity{
			CategoryCodes: []string{},
		},
	}
	checkInTask := sess.CheckIn(ctx, checkInConfig)
	awaitTask(checkInTask)
	checkInResult := runtimex.Try1(checkInTask.Result())
	log.Infof("%+v", checkInResult)

	// measure and submit all the URLs
	runtimex.Assert(checkInResult.Tests.WebConnectivity != nil, "nil WebConnectivity")
	for _, entry := range checkInResult.Tests.WebConnectivity.URLs {
		// TODO(bassosimone): the only problem with this style of measurement
		// is that we create a new report ID for each measurement
		//
		// There are two options here:
		//
		// 1. we create a new Experiment explicitly
		//
		// 2. we change the way in which we determine whether to open a new report
		//
		// I think the first option is way better than the second.

		// perform the measurement
		measurementTask := sess.Measure(
			ctx,
			"web_connectivity@v0.5",
			entry.URL,
			make(map[string]any),
		)
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
