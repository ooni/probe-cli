package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
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
	m := NewExperimentMeasurer(Config{})
	ctx := context.Background()
	measurement := &model.Measurement{}
	callbacks := model.NewPrinterCallbacks(log.Log)
	sess := &mockable.Session{
		MockableLogger:  log.Log,
		MockableTempDir: tempdir,
	}
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err = m.Run(ctx, args); err != nil {
		log.WithError(err).Fatal("torsf experiment failed")
	}
	data, err := json.Marshal(measurement)
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	fmt.Printf("%s\n", data)
}
