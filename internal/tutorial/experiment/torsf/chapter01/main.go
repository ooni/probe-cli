package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/torsf"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"golang.org/x/sys/execabs"
)

func main() {
	if _, err := execabs.LookPath("tor"); err != nil {
		log.Fatal("cannot find the tor executable in path")
	}
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.WithError(err).Fatal("cannot create temporary directory")
	}
	m := torsf.NewExperimentMeasurer(torsf.Config{})
	ctx := context.Background()
	measurement := &model.Measurement{}
	callbacks := model.NewPrinterCallbacks(log.Log)
	sess := &mockable.Session{
		MockableLogger:  log.Log,
		MockableTempDir: tempdir,
	}
	if err = m.Run(ctx, sess, measurement, callbacks); err != nil {
		log.WithError(err).Fatal("torsf experiment failed")
	}
	data, err := json.Marshal(measurement.TestKeys)
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	fmt.Printf("%s\n", data)
}
