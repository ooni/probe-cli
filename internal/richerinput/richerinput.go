// Package richerinput implements richer input masurements.
package richerinput

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Experiment is an experiment using richer input.
type Experiment[Config any] struct {
	// bc is the byte counter.
	bc *bytecounter.Counter

	// cbs contains experiment callbacks.
	cbs model.ExperimentCallbacks

	// factory constructs a new experiment measurer.
	factory func(config Config) model.ExperimentMeasurer

	// sess is the experiment session to use.
	sess model.RicherInputSession

	// testName contains the test name.
	testName string

	// testStartTime contains the test start time.
	testStartTime time.Time

	// testVersion contains the test version.
	testVersion string
}

// NewExperiment constructs a new [*Experiment] instance.
func NewExperiment[Config any](
	cbs model.ExperimentCallbacks,
	session model.RicherInputSession,
	testName string,
	testVersion string,
	factory func(config Config) model.ExperimentMeasurer,
) *Experiment[Config] {
	return &Experiment[Config]{
		bc:            bytecounter.New(),
		cbs:           cbs,
		factory:       factory,
		sess:          session,
		testName:      testName,
		testStartTime: time.Now().UTC(),
		testVersion:   testVersion,
	}
}

var _ model.RicherInputExperiment = &Experiment[int]{}

// KibiBytesReceived implements [model.RicherInputExperiment].
func (r *Experiment[Config]) KibiBytesReceived() float64 {
	return r.bc.KibiBytesReceived()
}

// KibiBytesSent implements [model.RicherInputExperiment].
func (r *Experiment[Config]) KibiBytesSent() float64 {
	return r.bc.KibiBytesSent()
}

// Measure implements [model.RicherInputExperiment].
func (r *Experiment[Config]) Measure(ctx context.Context, input model.RicherInput) (*model.Measurement, error) {
	// upgrade the context to use the experiment byte counter
	ctx = bytecounter.WithExperimentByteCounter(ctx, r.bc)

	// parse the configuration from JSON
	var config Config
	if err := json.Unmarshal(input.Options, &config); err != nil {
		return nil, err
	}

	// create a measurer instance
	mx := r.factory(config)

	// create the measurement
	measurement := r.newMeasurement(input, config)

	// create measurer args
	args := &model.ExperimentArgs{
		Callbacks:   r.cbs,
		Measurement: measurement,
		Session:     r.sess,
	}

	// save the time before running
	start := time.Now()

	// perform the measurement proper
	err := mx.Run(ctx, args)

	// save the time after running
	stop := time.Now()

	// handle the error case
	if err != nil {
		return nil, err
	}

	// compute the measurement runtime
	measurement.MeasurementRuntime = stop.Sub(start).Seconds()

	// scrub the measurement
	if err := model.ScrubMeasurement(measurement, r.sess.ProbeIP()); err != nil {
		return nil, err
	}

	// handle the successful case
	return measurement, nil
}

// NewReportTemplate implements [model.RicherInputExperiment].
func (r *Experiment[Config]) NewReportTemplate() *model.OOAPIReportTemplate {
	return &model.OOAPIReportTemplate{
		DataFormatVersion: model.OOAPIReportDefaultDataFormatVersion,
		Format:            model.OOAPIReportDefaultFormat,
		ProbeASN:          r.sess.ProbeASNString(),
		ProbeCC:           r.sess.ProbeCC(),
		SoftwareName:      r.sess.SoftwareName(),
		SoftwareVersion:   r.sess.SoftwareVersion(),
		TestName:          r.testName,
		TestStartTime:     r.testStartTimeString(),
		TestVersion:       r.testVersion,
	}
}

// testStartTimeString is a convenience method to get the test start time as a string.
func (r *Experiment[Config]) testStartTimeString() string {
	return r.testStartTime.Format(model.MeasurementDateFormat)
}

// newMeasurement creates a new measurement for this experiment with the given input.
func (r *Experiment[Config]) newMeasurement(input model.RicherInput, config Config) *model.Measurement {
	// get the current time used to init the measurement
	utctimenow := time.Now().UTC()

	// fill all the fields we can fill now
	m := &model.Measurement{
		Annotations:               map[string]string{}, // set below
		DataFormatVersion:         model.OOAPIReportDefaultDataFormatVersion,
		Extensions:                map[string]int64{}, // set by the experiment
		ID:                        "",                 // unused
		Input:                     model.MeasurementTarget(input.Input),
		InputHashes:               []string{}, // unused
		MeasurementStartTime:      utctimenow.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: utctimenow,
		Options:                   configToStringList(config),
		ProbeASN:                  r.sess.ProbeASNString(),
		ProbeCC:                   r.sess.ProbeCC(),
		ProbeCity:                 "", // unused
		ProbeIP:                   model.DefaultProbeIP,
		ProbeNetworkName:          r.sess.ProbeNetworkName(),
		ReportID:                  "", // set when we're submitting
		ResolverASN:               r.sess.ResolverASNString(),
		ResolverIP:                r.sess.ResolverIP(),
		ResolverNetworkName:       r.sess.ResolverNetworkName(),
		SoftwareName:              r.sess.SoftwareName(),
		SoftwareVersion:           r.sess.SoftwareVersion(),
		TestHelpers:               nil, // set by the experiment
		TestKeys:                  nil, // set by the experiment
		TestName:                  r.testName,
		MeasurementRuntime:        0, // set after running
		TestStartTime:             r.testStartTimeString(),
		TestVersion:               r.testVersion,
	}

	// add all the user-defined annotations first
	m.AddAnnotations(input.Annotations)

	// then add all the standard annotations
	m.AddAnnotation("architecture", runtime.GOARCH)
	m.AddAnnotation("engine_name", "ooniprobe-engine")
	m.AddAnnotation("engine_version", version.Version)
	m.AddAnnotation("go_version", runtimex.BuildInfo.GoVersion)
	m.AddAnnotation("platform", r.sess.Platform())
	m.AddAnnotation("vcs_modified", runtimex.BuildInfo.VcsModified)
	m.AddAnnotation("vcs_revision", runtimex.BuildInfo.VcsRevision)
	m.AddAnnotation("vcs_time", runtimex.BuildInfo.VcsTime)
	m.AddAnnotation("vcs_tool", runtimex.BuildInfo.VcsTool)

	return m
}

func configToStringList[Config any](config Config) (output []string) {
	// obtain the config struct value
	structValue := reflect.ValueOf(config)
	if structValue.Kind() == reflect.Pointer {
		structValue = structValue.Elem()
	}
	if structValue.Kind() != reflect.Struct {
		return
	}

	// then obtain the config struct type
	structType := structValue.Type()

	// include all the possible fields
	for idx := 0; idx < structType.NumField(); idx++ {
		// obtain field value
		fieldValue := structValue.Field(idx)

		// obtain field type
		fieldType := structType.Field(idx)

		// ignore fields that are not exported
		if !fieldType.IsExported() {
			continue
		}

		// ignore fields whose name starts with "Safe"
		if strings.HasPrefix(fieldType.Name, "Safe") {
			continue
		}

		// append the field value
		output = append(output, fmt.Sprintf("%s=%v", fieldType.Name, fieldValue.Interface()))
	}
	return
}
