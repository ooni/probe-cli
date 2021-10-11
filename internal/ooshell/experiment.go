package ooshell

//
// experiment.go
//
// Code to run experiments.
//

import (
	"context"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// experimentDB is an engine.Experiment with knowledge about the DB.
type experimentDB struct {
	// db is the DB to use
	db DB

	// env is the underlying Environ.
	env *Environ

	// id is the experiment ID.
	id int64

	// index is the experiment index in the group.
	index int

	// logger is the logger to use.
	logger Logger

	// name is the name of the experiment.
	name string

	// result is the result to which we belong.
	result *resultDB

	// sess is the underlying measurement session.
	sess *sessionDB

	// total is the number of experiments inside the group.
	total int
}

// newExperimentDB creates a new experimentDB instance.
func (r *resultDB) newExperimentDB(name string, idx, total int) (*experimentDB, error) {
	experimentID, err := r.db.NewExperiment(r.id, name)
	if err != nil {
		return nil, err
	}
	return &experimentDB{
		db:     r.db,
		env:    r.env,
		id:     experimentID,
		index:  idx,
		logger: r.env.Logger,
		name:   name,
		result: r,
		sess:   r.sess,
		total:  total,
	}, nil
}

// Run runs the experiment.
func (exp *experimentDB) Run(ctx context.Context) error {
	builder, err := exp.sess.NewExperimentBuilder(exp.name)
	if err != nil {
		return err
	}
	exp.setExperimentProgressCb(builder)
	if err := exp.addOptions(builder); err != nil {
		return err
	}
	annotations, err := exp.parseAnnotations()
	if err != nil {
		return err
	}
	exp.logger.Infof("loading inputs; please, be patient...")
	inputs, err := exp.loadInputs(ctx, builder)
	// TODO(bassosimone): record loading inputs errors?
	if err != nil {
		return err
	}
	submitter, err := engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: !exp.env.NoCollector,
		Session: exp.sess,
		Logger:  exp.logger,
	})
	if err != nil {
		return err
	}
	experiment := builder.NewExperiment()
	saver, err := engine.NewSaver(engine.SaverConfig{
		Enabled:    !exp.env.NoJSON,
		Experiment: experiment,
		FilePath:   exp.env.ReportFile,
		Logger:     exp.logger,
	})
	if err != nil {
		return err
	}
	// TODO(bassosimone): fix this
	/*
		defer func() {
			received := experiment.KibiBytesReceived()
			sent := experiment.KibiBytesSent()
			env.Callbacks.OnDataUsage(name, received, sent)
		}()
	*/
	return exp.loop(
		ctx, submitter, saver, experiment, inputs, annotations, builder.InputPolicy())
}

// setExperimentProgressCb sets the correct experiment progress callback
// depending on whether the experiment takes input or not.
//
// If the experiment takes input, we're going to emit progress based
// either on the maximum runtime or the number of left URLs.
//
// Otherwise, we'll forward the experiment progress events to the controller.
func (exp *experimentDB) setExperimentProgressCb(builder *engine.ExperimentBuilder) {
	switch builder.InputPolicy() {
	case engine.InputNone:
		builder.SetCallbacks(&callbacksReportBack{exp})
	default:
		builder.SetCallbacks(&callbacksNull{})
	}
}

// loop runs the experiment loop.
func (exp *experimentDB) loop(ctx context.Context, submitter engine.Submitter,
	saver engine.Saver, experiment *engine.Experiment, inputs []model.URLInfo,
	annotations map[string]string, inputPolicy engine.InputPolicy) error {
	start := time.Now()
	maxRuntime := time.Duration(exp.env.MaxRuntime) * time.Second
	for idx, url := range inputs {
		// TODO(bassosimone): interrupt also using the context here
		if maxRuntime > 0 && time.Since(start) > maxRuntime {
			return nil
		}
		input := url.URL
		exp.maybeEmitProgress(experiment.Name(),
			inputPolicy, idx, len(inputs), maxRuntime, start, input)
		source, err := experiment.MeasureAsync(ctx, input)
		if err != nil {
			return err
		}
		// NOTE: we don't want to intermix measuring with submitting
		// therefore we first gather all the measurements and then submit
		err = exp.msubmit(ctx, submitter, saver, &url, annotations, exp.mgather(source))
		if err != nil {
			return err
		}
	}
	return nil
}

// maybeEmitProgress emits progress if this experiment takes input. The
// emitted progress depends on whether we're using maxRuntime.
func (exp *experimentDB) maybeEmitProgress(name string, inputPolicy engine.InputPolicy,
	idx, total int, maxRuntime time.Duration, start time.Time, input string) {
	if inputPolicy == engine.InputNone {
		return
	}
	var ratio float64
	if maxRuntime <= 0 && total > 0 {
		ratio = float64(idx) / float64(total)
	} else if maxRuntime > 0 {
		elapsed := time.Since(start)
		ratio = float64(elapsed) / float64(maxRuntime)
	}
	if ratio > 1.0 {
		ratio = 1.0
	}
	exp.handleProgress(ratio, fmt.Sprintf("processing %s", input))
}

// handleProgress is called by callbacksReportBack and maybeEmitProgress
// to actually pass progress information to the database.
func (exp *experimentDB) handleProgress(percentage float64, message string) {
	step := 1 / float64(exp.total)
	percentage *= step
	percentage += float64(exp.index) * step
	exp.db.UpdateExperimentProgress(exp.id, percentage, message)
}

// mgather gathers all measurements.
func (exp *experimentDB) mgather(in <-chan *model.Measurement) (out []*model.Measurement) {
	for meas := range in {
		out = append(out, meas)
	}
	return
}

// msubmit submits all measurements.
func (exp *experimentDB) msubmit(
	ctx context.Context, submitter engine.Submitter, saver engine.Saver,
	urlInfo *model.URLInfo, annotations map[string]string,
	measurements []*model.Measurement) error {
	for _, meas := range measurements {
		meas.AddAnnotations(annotations)
		meas.Options = exp.env.Options
		if err := submitter.Submit(ctx, meas); err != nil {
			// nothing for now
		}
		// Note: must be after submission because submission modifies
		// the measurement to include the report ID.
		if err := saver.SaveMeasurement(meas); err != nil {
			// nothing for now
		}
	}
	return nil
}
