// Package example contains a simple example experiment.
//
// You could use this code to boostrap the implementation of
// a new experiment that you are working on.
package example

import (
	"context"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const testVersion = "0.1.0"

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
type Config struct {
	Message     string `ooni:"Message to emit at test completion"`
	ReturnError bool   `ooni:"Toogle to return a mocked error"`
	SleepTime   int64  `ooni:"Amount of time to sleep for"`
}

// TestKeys contains the experiment's result.
//
// This is what will end up into the Measurement.TestKeys field
// when you run this experiment.
//
// In other words, the variables in this struct will be
// the specific results of this experiment.
type TestKeys struct {
	Success bool `json:"success"`
}

// Measurer performs the measurement.
type Measurer struct {
	config   Config
	testName string
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return m.testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// ErrFailure is the error returned when you set the
// config.ReturnError field to true.
var ErrFailure = errors.New("mocked error")

// Run implements model.ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session
	var err error
	if m.config.ReturnError {
		err = ErrFailure
	}
	testkeys := &TestKeys{Success: err == nil}
	measurement.TestKeys = testkeys
	sess.Logger().Warnf("%s", "Follow the white rabbit.")
	ctx, cancel := context.WithTimeout(ctx, time.Duration(m.config.SleepTime))
	defer cancel()
	<-ctx.Done()
	sess.Logger().Infof("%s", "Knock, knock, Neo.")
	callbacks.OnProgress(1.0, m.config.Message)
	// Note: if here we return an error, the parent code will assume
	// something fundamental was wrong and we don't have a measurement
	// to submit to the OONI collector. Keep this in mind when you
	// are writing new experiments!
	return err
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config, testName string) model.ExperimentMeasurer {
	return Measurer{config: config, testName: testName}
}
