package oonirun

//
// Run experiments.
//

import (
	"context"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
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

	// ExtraOptions contains OPTIONAL extra options for the experiment.
	ExtraOptions map[string]any

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

	// newInputLoaderFn is OPTIONAL and used for testing.
	newInputLoaderFn func(inputPolicy model.InputPolicy) inputLoader

	// newSubmitterFn is OPTIONAL and used for testing.
	newSubmitterFn func(ctx context.Context) (model.Submitter, error)

	// newSaverFn is OPTIONAL and used for testing.
	newSaverFn func(experiment model.RicherInputExperiment) (model.Saver, error)

	// newInputProcessorFn is OPTIONAL and used for testing.
	newInputProcessorFn func(experiment model.RicherInputExperiment, inputList []model.RicherInput,
		saver model.Saver, submitter model.Submitter) inputProcessor
}

// Run runs the given experiment.
func (ed *Experiment) Run(ctx context.Context) error {

	// 1. create experiment builder
	builder, err := ed.newExperimentBuilder(ed.Name)
	if err != nil {
		return err
	}

	// 2. create input loader and load input for this experiment
	inputLoader := ed.newInputLoader(builder.InputPolicy())
	inputList, err := inputLoader.Load(ctx)
	if err != nil {
		return err
	}

	// 3. randomize input, if needed
	if ed.Random {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404 -- not really important
		rnd.Shuffle(len(inputList), func(i, j int) {
			inputList[i], inputList[j] = inputList[j], inputList[i]
		})
		experimentShuffledInputs.Add(1)
	}

	// 4. configure experiment's options
	if err := builder.SetOptionsAny(ed.ExtraOptions); err != nil {
		return err
	}

	// 5. construct the experiment instance
	experiment := builder.NewRicherInputExperiment()
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
	saver, err := ed.newSaver(experiment)
	if err != nil {
		return err
	}

	// 8. convert the API input list to richer input
	richerInputList := builder.BuildRicherInput(
		ed.Annotations,
		experimentOOAPIURLInfoToFlatInputList(inputList...),
	)

	// 9. create an input processor
	inputProcessor := ed.newInputProcessor(experiment, richerInputList, saver, submitter)

	// 10. process input and generate measurements
	return inputProcessor.Run(ctx)
}

func experimentOOAPIURLInfoToFlatInputList(inputs ...model.OOAPIURLInfo) (outputs []string) {
	for _, input := range inputs {
		outputs = append(outputs, input.URL)
	}
	return
}

// inputProcessor is an alias for model.ExperimentInputProcessor
type inputProcessor = model.ExperimentInputProcessor

// newInputProcessor creates a new inputProcessor instance.
func (ed *Experiment) newInputProcessor(experiment model.RicherInputExperiment,
	inputList []model.RicherInput, saver model.Saver, submitter model.Submitter) inputProcessor {
	if ed.newInputProcessorFn != nil {
		return ed.newInputProcessorFn(experiment, inputList, saver, submitter)
	}
	return &InputProcessor{
		Experiment: &experimentWrapper{
			child:  NewInputProcessorExperimentWrapper(experiment),
			logger: ed.Session.Logger(),
			total:  len(inputList),
		},
		Inputs:     inputList,
		MaxRuntime: time.Duration(ed.MaxRuntime) * time.Second,
		Saver:      NewInputProcessorSaverWrapper(saver),
		Submitter: &experimentSubmitterWrapper{
			child:  NewInputProcessorSubmitterWrapper(submitter),
			logger: ed.Session.Logger(),
		},
	}
}

// newSaver creates a new engine.Saver instance.
func (ed *Experiment) newSaver(experiment model.RicherInputExperiment) (model.Saver, error) {
	if ed.newSaverFn != nil {
		return ed.newSaverFn(experiment)
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

// inputLoader is an alias for model.ExperimentInputLoader
type inputLoader = model.ExperimentInputLoader

// newInputLoader creates a new inputLoader.
func (ed *Experiment) newInputLoader(inputPolicy model.InputPolicy) inputLoader {
	if ed.newInputLoaderFn != nil {
		return ed.newInputLoaderFn(inputPolicy)
	}
	return &engine.InputLoader{
		CheckInConfig: &model.OOAPICheckInConfig{
			RunType:  model.RunTypeManual,
			OnWiFi:   true, // meaning: not on 4G
			Charging: true,
		},
		ExperimentName: ed.Name,
		InputPolicy:    inputPolicy,
		StaticInputs:   ed.Inputs,
		SourceFiles:    ed.InputFilePaths,
		Session:        ed.Session,
	}
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

func (ew *experimentWrapper) Measure(
	ctx context.Context, input model.RicherInput, idx int) (*model.Measurement, error) {
	if input.Input != "" {
		ew.logger.Infof("[%d/%d] running with input: %s", idx+1, ew.total, input)
	}
	return ew.child.Measure(ctx, input, idx)
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
