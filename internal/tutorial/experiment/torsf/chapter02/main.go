package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
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
	// -=-=- StartHere -=-=-
	//
	// # Chapter II: creating an empty experiment
	//
	// In this chapter we will create an empty experiment and replace
	// the code calling the real `torsf` experiment in `main.go` to
	// call our empty experiment instead.
	//
	// (This file is auto-generated from the corresponding source file,
	// so make sure you don't edit it manually.)
	//
	// ## Changes in main.go
	//
	// In `main.go` we will simply replace the call to the
	// `torsf.NewExperimentMeasurer` function with a call to
	// a `NewExperimentMeasurer` function that we are going
	// to implement as part of this chapter.
	//
	// After you do this, you also need to remove the now-unneded
	// import of the `torsf` package.
	//
	// There are no additional changes to `main.go`.
	//
	// ```Go
	m := NewExperimentMeasurer(Config{})
	// ```
	// -=-=- StopHere -=-=-
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
