package oonirun

//
// Run experiments.
//

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// experimentShuffledInputs counts how many times we shuffled inputs
var experimentShuffledInputs = &atomic.Int64{}

// Experiment describes an experiment to run. You MUST fill all the fields that
// are marked as MANDATORY, otherwise Experiment.Run will cause panics.
type Experiment struct {
	// Annotations contains OPTIONAL Annotations for the experiment.
	Annotations map[string]string

	// ExtraOptions contains OPTIONAL extra options that modify the
	// default experiment-specific configuration. We apply
	// the changes described by this field after using the InitialOptions
	// field to initialize the experiment-specific configuration.
	ExtraOptions map[string]any

	// InitialOptions contains an OPTIONAL [json.RawMessage] object
	// used to initialize the default experiment-specific
	// configuration. After we have initialized the configuration
	// as such, we then apply the changes described by the ExtraOptions.
	InitialOptions json.RawMessage

	// Inputs contains the OPTIONAL experiment Inputs
	Inputs []string

	// InputFilePaths contains OPTIONAL files to read inputs from.
	InputFilePaths []string

	// MaxRuntime is the OPTIONAL maximum runtime in seconds.
	MaxRuntime int64

	// Name is the MANDATORY experiment name.
	Name string

	// NoCollector OPTIONALLY indicates we should not be using any collector.
	NoCollector bool

	// NoJSON OPTIONALLY indicates we don't want to save measurements to a JSON file.
	NoJSON bool

	// Random OPTIONALLY indicates we should randomize inputs.
	Random bool

	// ReportFile is the MANDATORY file in which to save reports, which is only
	// used when noJSON is set to false.
	ReportFile string

	// Session is the MANDATORY session.
	Session Session

	// newExperimentBuilderFn is OPTIONAL and used for testing.
	newExperimentBuilderFn func(experimentName string) (model.ExperimentBuilder, error)

	// newTargetLoaderFn is OPTIONAL and used for testing.
	newTargetLoaderFn func(builder model.ExperimentBuilder) targetLoader

	// newSubmitterFn is OPTIONAL and used for testing.
	newSubmitterFn func(ctx context.Context) (model.Submitter, error)

	// newSaverFn is OPTIONAL and used for testing.
	newSaverFn func() (model.Saver, error)

	// newInputProcessorFn is OPTIONAL and used for testing.
	newInputProcessorFn func(experiment model.Experiment, inputList []model.ExperimentTarget,
		saver model.Saver, submitter model.Submitter) inputProcessor
}

// Run runs the given experiment.
func (ed *Experiment) Run(ctx context.Context) error {

	// 1. create experiment builder
	builder, err := ed.newExperimentBuilder(ed.Name)
	if err != nil {
		return err
	}

	// TODO(bassosimone): we need another patch after the current one
	// to correctly serialize the options as configured using InitialOptions
	// and ExtraOptions otherwise the Measurement.Options field turns out
	// to always be empty and this is highly suboptimal for us.
	//
	// The next patch is https://github.com/ooni/probe-cli/pull/1630.

	// 2. configure experiment's options
	//
	// This MUST happen before loading targets because the options will
	// possibly be used to produce richer input targets.
	if err := ed.setOptions(builder); err != nil {
		return err
	}

	// 3. create target loader and load targets for this experiment
	targetLoader := ed.newTargetLoader(builder)
	targetList, err := targetLoader.Load(ctx)
	if err != nil {
		return err
	}

	// 4. randomize input, if needed
	if ed.Random {
		// Note: since go1.20 the default random generated is random seeded
		//
		// See https://tip.golang.org/doc/go1.20
		rand.Shuffle(len(targetList), func(i, j int) {
			targetList[i], targetList[j] = targetList[j], targetList[i]
		})
		experimentShuffledInputs.Add(1)
	}

	// 5. construct the experiment instance
	experiment := builder.NewExperiment()
	logger := ed.Session.Logger()
	defer func() {
		logger.Infof("experiment: recv %s, sent %s",
			humanize.SI(experiment.KibiBytesReceived()*1024, "byte"),
			humanize.SI(experiment.KibiBytesSent()*1024, "byte"),
		)
	}()

	// 6. create the submitter
	submitter, err := ed.newSubmitter(ctx)
	if err != nil {
		return err
	}

	// 7. create the saver
	saver, err := ed.newSaver()
	if err != nil {
		return err
	}

	// 8. create an input processor
	inputProcessor := ed.newInputProcessor(experiment, targetList, saver, submitter)

	// 9. process input and generate measurements
	return inputProcessor.Run(ctx)
}

func (ed *Experiment) setOptions(builder model.ExperimentBuilder) error {
	// We first unmarshal the InitialOptions into the experiment
	// configuration and afterwards we modify the configuration using
	// the values contained inside the ExtraOptions field.
	if err := builder.SetOptionsJSON(ed.InitialOptions); err != nil {
		return err
	}
	return builder.SetOptionsAny(ed.ExtraOptions)
}

// inputProcessor is an alias for model.ExperimentInputProcessor
type inputProcessor = model.ExperimentInputProcessor

// newInputProcessor creates a new inputProcessor instance.
func (ed *Experiment) newInputProcessor(experiment model.Experiment,
	inputList []model.ExperimentTarget, saver model.Saver, submitter model.Submitter) inputProcessor {
	if ed.newInputProcessorFn != nil {
		return ed.newInputProcessorFn(experiment, inputList, saver, submitter)
	}
	return &InputProcessor{
		Annotations: ed.Annotations,
		Experiment: &experimentWrapper{
			child:  NewInputProcessorExperimentWrapper(experiment),
			logger: ed.Session.Logger(),
			total:  len(inputList),
		},
		Inputs:     inputList,
		MaxRuntime: time.Duration(ed.MaxRuntime) * time.Second,
		Options:    experimentOptionsToStringList(ed.ExtraOptions),
		Saver:      NewInputProcessorSaverWrapper(saver),
		Submitter: &experimentSubmitterWrapper{
			child:  NewInputProcessorSubmitterWrapper(submitter),
			logger: ed.Session.Logger(),
		},
	}
}

// newSaver creates a new engine.Saver instance.
func (ed *Experiment) newSaver() (model.Saver, error) {
	if ed.newSaverFn != nil {
		return ed.newSaverFn()
	}
	return NewSaver(SaverConfig{
		Enabled:  !ed.NoJSON,
		FilePath: ed.ReportFile,
		Logger:   ed.Session.Logger(),
	})
}

// newSubmitter creates a new engine.Submitter instance.
func (ed *Experiment) newSubmitter(ctx context.Context) (model.Submitter, error) {
	if ed.newSubmitterFn != nil {
		return ed.newSubmitterFn(ctx)
	}
	return NewSubmitter(ctx, SubmitterConfig{
		Enabled: !ed.NoCollector,
		Session: ed.Session,
		Logger:  ed.Session.Logger(),
	})
}

// newExperimentBuilder creates a new engine.ExperimentBuilder for the given experimentName.
func (ed *Experiment) newExperimentBuilder(experimentName string) (model.ExperimentBuilder, error) {
	if ed.newExperimentBuilderFn != nil {
		return ed.newExperimentBuilderFn(experimentName)
	}
	return ed.Session.NewExperimentBuilder(ed.Name)
}

// targetLoader is an alias for [model.ExperimentTargetLoader].
type targetLoader = model.ExperimentTargetLoader

// newTargetLoader creates a new [model.ExperimentTargetLoader].
func (ed *Experiment) newTargetLoader(builder model.ExperimentBuilder) targetLoader {
	if ed.newTargetLoaderFn != nil {
		return ed.newTargetLoaderFn(builder)
	}
	return builder.NewTargetLoader(&model.ExperimentTargetLoaderConfig{
		CheckInConfig: &model.OOAPICheckInConfig{
			RunType:  model.RunTypeManual,
			OnWiFi:   true, // meaning: not on 4G
			Charging: true,
		},
		StaticInputs: ed.Inputs,
		SourceFiles:  ed.InputFilePaths,
		Session:      ed.Session,
	})
}

// experimentOptionsToStringList convers the options to []string, which is
// the format with which we include them into a OONI Measurement. The resulting
// []string will skip any option that is named with a `Safe` prefix (case
// sensitive).
func experimentOptionsToStringList(options map[string]any) (out []string) {
	// the prefix to skip inclusion in the string list
	safeOptionPrefix := "Safe"
	for key, value := range options {
		if strings.HasPrefix(key, safeOptionPrefix) {
			continue
		}
		out = append(out, fmt.Sprintf("%s=%v", key, value))
	}
	return
}

// experimentWrapper wraps an experiment and logs progress
type experimentWrapper struct {
	// child is the child experiment wrapper
	child InputProcessorExperimentWrapper

	// logger is the logger to use
	logger model.Logger

	// total is the total number of inputs
	total int
}

func (ew *experimentWrapper) MeasureWithContext(
	ctx context.Context, target model.ExperimentTarget, idx int) (*model.Measurement, error) {
	if target.Input() != "" {
		ew.logger.Infof("[%d/%d] running with input: %s", idx+1, ew.total, target)
	}
	return ew.child.MeasureWithContext(ctx, target, idx)
}

// experimentSubmitterWrapper implements a submission policy where we don't
// fail if we cannot submit a measurement
type experimentSubmitterWrapper struct {
	// child is the child submitter wrapper
	child InputProcessorSubmitterWrapper

	// logger is the logger to use
	logger model.Logger
}

func (sw *experimentSubmitterWrapper) Submit(ctx context.Context, idx int, m *model.Measurement) error {
	if err := sw.child.Submit(ctx, idx, m); err != nil {
		sw.logger.Warnf("submitting measurement failed: %s", err.Error())
	}
	// policy: we do not stop the loop if measurement submission fails
	return nil
}
