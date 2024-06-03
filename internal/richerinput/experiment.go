package richerinput

//
// Implementation of the richer input experiment
//

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/erroror"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Experiment is a richer-input-aware experiment.
type Experiment[T Target] struct {
	// byteCounter is the byte counter.
	byteCounter *bytecounter.Counter

	// inputLoader loads richer input for this experiment.
	inputLoader InputLoader[T]

	// measurer is the underlying measurer.
	measurer Measurer[T]

	// mu provides mutual exclusion when accessing fields.
	mu sync.Mutex

	// report is the possibly nil report we're using.
	report optional.Value[model.OOAPIReport]

	// sess is the MANDATORY session we're using.
	sess model.RicherInputSession

	// startTimeUTC is the start time of this experiment using UTC.
	startTimeUTC time.Time
}

var _ model.RicherInputExperiment = &Experiment[VoidTarget]{}

// NewExperiment creates a new [*Experiment] instance.
func NewExperiment[T Target](
	inputLoader InputLoader[T], measurer Measurer[T], session model.RicherInputSession) *Experiment[T] {
	return &Experiment[T]{
		byteCounter:  bytecounter.New(),
		inputLoader:  inputLoader,
		measurer:     measurer,
		mu:           sync.Mutex{},
		report:       optional.None[model.OOAPIReport](),
		sess:         session,
		startTimeUTC: time.Now().UTC(),
	}
}

// KibiBytesReceived implements model.RicherInputExperiment.
func (e *Experiment[T]) KibiBytesReceived() float64 {
	return e.byteCounter.KibiBytesReceived()
}

// KibiBytesSent implements model.RicherInputExperiment.
func (e *Experiment[T]) KibiBytesSent() float64 {
	return e.byteCounter.KibiBytesSent()
}

// Name implements model.RicherInputExperiment.
func (e *Experiment[T]) Name() string {
	return e.measurer.ExperimentName()
}

// OpenReport implements model.RicherInputExperiment.
func (e *Experiment[T]) OpenReport(ctx context.Context) error {
	// run in mutal exclusion
	defer e.mu.Unlock()
	e.mu.Lock()

	// attempt to open a report, if needed
	if e.report.IsNone() {
		report, err := e.sess.OpenReport(ctx, e.NewReportTemplate())
		if err != nil {
			return err
		}
		e.report = optional.Some(report)
	}

	return nil
}

// NewReportTemplate constructs a new [*model.OOAPIReportTemplate] for this experiment.
func (e *Experiment[T]) NewReportTemplate() *model.OOAPIReportTemplate {
	return &model.OOAPIReportTemplate{
		DataFormatVersion: model.OOAPIReportDefaultDataFormatVersion,
		Format:            model.OOAPIReportDefaultFormat,
		ProbeASN:          e.sess.ProbeASNString(),
		ProbeCC:           e.sess.ProbeCC(),
		SoftwareName:      e.sess.SoftwareName(),
		SoftwareVersion:   e.sess.SoftwareVersion(),
		TestName:          e.measurer.ExperimentName(),
		TestStartTime:     e.startTimeUTC.Format(model.MeasurementDateFormat),
		TestVersion:       e.measurer.ExperimentVersion(),
	}
}

// NewMeasurement creates a new measurement for this experiment with the given input.
func (e *Experiment[T]) NewMeasurement(config *model.RicherInputConfig, target T) *model.Measurement {
	// compute the current time in UTC to fill the measurement
	utctimenow := time.Now().UTC()

	// fill the fiels we can immediately fill
	m := &model.Measurement{
		Annotations:               map[string]string{},
		DataFormatVersion:         model.OOAPIReportDefaultDataFormatVersion,
		Extensions:                map[string]int64{}, // filled by the experiment measurer
		ID:                        "",                 // unused
		Input:                     target.Input(),
		InputHashes:               nil, // unused
		MeasurementStartTime:      utctimenow.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: utctimenow,
		Options:                   target.Options(),
		ProbeASN:                  e.sess.ProbeASNString(),
		ProbeCC:                   e.sess.ProbeCC(),
		ProbeCity:                 "", // unused
		ProbeIP:                   model.DefaultProbeIP,
		ProbeNetworkName:          e.sess.ProbeNetworkName(),
		ReportID:                  e.ReportID(),
		ResolverASN:               e.sess.ResolverASNString(),
		ResolverIP:                e.sess.ResolverIP(),
		ResolverNetworkName:       e.sess.ResolverNetworkName(),
		SoftwareName:              e.sess.SoftwareName(),
		SoftwareVersion:           e.sess.SoftwareVersion(),
		TestHelpers:               nil, // set by the experiment measurer
		TestKeys:                  nil, // ditto
		TestName:                  e.measurer.ExperimentName(),
		MeasurementRuntime:        0, // set after the measurement has finished
		TestStartTime:             e.startTimeUTC.Format(model.MeasurementDateFormat),
		TestVersion:               e.measurer.ExperimentVersion(),
	}

	// add all the user-defined annotations
	m.AddAnnotations(config.Annotations)

	// TODO(bassosimone): maybe the "add" semantic is not clear enough and it
	// would be more clear if the methods were named "SetAnnotation{,s}"?

	// then, add the standard annotations
	//
	// this guarantees that a user cannot override standard annotations
	m.AddAnnotation("architecture", runtime.GOARCH)
	m.AddAnnotation("engine_name", "ooniprobe-engine")
	m.AddAnnotation("engine_version", version.Version)
	m.AddAnnotation("go_version", runtimex.BuildInfo.GoVersion)
	m.AddAnnotation("platform", e.sess.Platform())
	m.AddAnnotation("vcs_modified", runtimex.BuildInfo.VcsModified)
	m.AddAnnotation("vcs_revision", runtimex.BuildInfo.VcsRevision)
	m.AddAnnotation("vcs_time", runtimex.BuildInfo.VcsTime)
	m.AddAnnotation("vcs_tool", runtimex.BuildInfo.VcsTool)

	return m
}

// ReportID implements model.RicherInputExperiment.
func (e *Experiment[T]) ReportID() (ID string) {
	// run in mutal exclusion
	defer e.mu.Unlock()
	e.mu.Lock()

	// access the report ID, if possible
	if report := e.report.UnwrapOr(nil); report != nil {
		ID = report.ReportID()
	}
	return
}

// Start implements model.RicherInputExperiment.
func (e *Experiment[T]) Start(
	ctx context.Context, config *model.RicherInputConfig) <-chan *erroror.Value[*model.Measurement] {
	output := make(chan *erroror.Value[*model.Measurement])
	go func() {
		defer close(output)
		e.run(ctx, config, output)
	}()
	return output
}

// run runs the experiment until completion.
func (e *Experiment[T]) run(
	ctx context.Context, config *model.RicherInputConfig, output chan<- *erroror.Value[*model.Measurement]) {
	// load richer input using the loader
	targets, err := e.inputLoader.Load(ctx, config)
	if err != nil {
		output <- &erroror.Value[*model.Measurement]{Err: err}
		return
	}

	// randomize input, if needed
	if config.RandomizeInputs {
		//		If Seed is not called, the generator is seeded randomly at program startup.
		//
		//		Prior to Go 1.20, the generator was seeded like Seed(1) at program startup. To force
		//		the old behavior, call Seed(1) at program startup. Alternately, set GODEBUG=randautoseed=0
		//		in the environment before making any calls to functions in this package.
		//
		// See https://pkg.go.dev/math/rand@go1.21.10#Seed
		rand.Shuffle(len(targets), func(i, j int) {
			targets[i], targets[j] = targets[j], targets[i]
		})
	}

	// let the caller know about the inputs we loaded
	//
	// note: this gives ooniprobe a chance to fill the URLs table
	// before we actually run the experiment
	e.onTargets(config, targets)

	// add maximum runtime iff we have more than one target
	//
	// note: it's undocumented but we clamp max runtime to a value that
	// should not lead to any multiplication overflow
	var deadline time.Time
	if config.MaxRuntime > 0 && len(targets) > 1 {
		maxRuntime := min(time.Duration(config.MaxRuntime), math.MaxInt64/time.Second)
		deadline = time.Now().Add(maxRuntime * time.Second)
	}

	// perform a measurement for each input
	//
	// note: we're not opening a report here and we leave the responsibility
	// of doing that for the current experiment to the Start caller
	for idx, target := range targets {
		// create new measurement
		measurement := e.NewMeasurement(config, target)

		// prepare arguments for the measurer
		args := &MeasurerRunArgs[T]{
			Callbacks:   config.Callbacks,
			Measurement: measurement,
			Session:     e.sess,
			Target:      target,
		}

		// save the time before starting to measure
		t0 := time.Now()

		// before measuring, handle the case where the max runtime has expired
		if !deadline.IsZero() && t0.Sub(deadline) > 0 {
			e.sess.Logger().Warnf("max runtime deadline expired")
			return
		}

		// emit progress about this target unless it's a VoidTarget instance
		if _, isVoidTarget := any(target).(VoidTarget); !isVoidTarget {
			progress := float64(idx) / float64(len(targets))
			config.Callbacks.OnProgress(progress, fmt.Sprintf("%v", target))
		}

		// invoke the measurer
		err := e.measurer.Run(ctx, args)

		// remember to fill the measurement runtime field
		measurement.MeasurementRuntime = time.Since(t0).Seconds()

		// handle the error case
		//
		// note: we keep looping on failure
		if err != nil {
			output <- &erroror.Value[*model.Measurement]{Err: err}
			continue
		}

		// TODO(bassosimone): we should scrub the measurement here

		// TODO(bassosimone): check whether we're missing anything else here

		// on success, emit the measurement
		output <- &erroror.Value[*model.Measurement]{Value: measurement}
	}
}

func (e *Experiment[T]) onTargets(config *model.RicherInputConfig, targets []T) {
	// Note: Go does not support passing a slice of a concrete type as a slice of a
	// generic type, so we need to create a copy of the original slice 😭
	//
	// Historical note: this reminds me of a comment that was part of tstat source
	// code and made me laugh when I was working on tstat in 2009:
	//
	//	@v = map { chomp } `cat $fname` #but C is stubborn
	//
	// See http://tstat.polito.it/viewvc/software/tstat/trunk/tstat/naivebayes.c?revision=965&view=markup
	var converted []model.RicherInputTarget
	for _, entry := range targets {
		converted = append(converted, entry)
	}
	config.Callbacks.OnTargets(converted)
}

// ErrReportIsNotOpen indicates that we have not opened a report yet.
var ErrReportIsNotOpen = errors.New("report is not open")

// SubmitMeasurement implements model.RicherInputExperiment.
func (e *Experiment[T]) SubmitMeasurement(ctx context.Context, m *model.Measurement) error {
	// run in mutal exclusion
	defer e.mu.Unlock()
	e.mu.Lock()

	// handle the case of report not being open
	report := e.report.UnwrapOr(nil)
	if report == nil {
		return ErrReportIsNotOpen
	}

	// attempt to submit
	return report.SubmitMeasurement(ctx, m)
}
