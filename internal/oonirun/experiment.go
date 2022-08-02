package oonirun

//
// Run experiments.
//

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/model"
)

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

	// Parallellism OPTIONALLY indicates the number of goroutines to use
	// to perform measurements. A zero or negative value implies one goroutine.
	Parallelism int

	// Random OPTIONALLY indicates we should randomize inputs.
	Random bool

	// ReportFile is the MANDATORY file in which to save reports, which is only
	// used when noJSON is set to false.
	ReportFile string

	// Session is the MANDATORY session.
	Session Session
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
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rnd.Shuffle(len(inputList), func(i, j int) {
			inputList[i], inputList[j] = inputList[j], inputList[i]
		})
	}

	// 4. configure experiment's options
	if err := builder.SetOptionsAny(ed.ExtraOptions); err != nil {
		return err
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
	saver, err := ed.newSaver(experiment)
	if err != nil {
		return err
	}

	// 8. create an input processor
	inputProcessor := ed.newInputProcessor(experiment, inputList, saver, submitter)

	// 9. process input and generate measurements
	return inputProcessor.Run(ctx)
}

// inputProcessor processes inputs running the given experiment.
type inputProcessor interface {
	Run(ctx context.Context) error
}

// newInputProcessor creates a new inputProcessor instance.
func (ed *Experiment) newInputProcessor(experiment engine.Experiment,
	inputList []model.OOAPIURLInfo, saver engine.Saver, submitter engine.Submitter) inputProcessor {
	return &engine.InputProcessor{
		Annotations: ed.Annotations,
		Experiment:  experiment,
		Inputs:      inputList,
		Logger:      ed.Session.Logger(),
		MaxRuntime:  time.Duration(ed.MaxRuntime) * time.Second,
		Options:     experimentOptionsToStringList(ed.ExtraOptions),
		Parallelism: ed.Parallelism,
		Saver:       saver,
		Submitter:   submitter,
	}
}

// newSaver creates a new engine.Saver instance.
func (ed *Experiment) newSaver(experiment engine.Experiment) (engine.Saver, error) {
	return engine.NewSaver(engine.SaverConfig{
		Enabled:    !ed.NoJSON,
		Experiment: experiment,
		FilePath:   ed.ReportFile,
		Logger:     ed.Session.Logger(),
	})
}

// newSubmitter creates a new engine.Submitter instance.
func (ed *Experiment) newSubmitter(ctx context.Context) (engine.Submitter, error) {
	return engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: !ed.NoCollector,
		Session: ed.Session,
		Logger:  ed.Session.Logger(),
	})
}

// newExperimentBuilder creates a new engine.ExperimentBuilder for the given experimentName.
func (ed *Experiment) newExperimentBuilder(experimentName string) (engine.ExperimentBuilder, error) {
	return ed.Session.NewExperimentBuilder(ed.Name)
}

// inputLoader loads inputs from local or remote sources.
type inputLoader interface {
	Load(ctx context.Context) ([]model.OOAPIURLInfo, error)
}

// newInputLoader creates a new inputLoader.
func (ed *Experiment) newInputLoader(inputPolicy engine.InputPolicy) inputLoader {
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

// experimentOptionsToStringList convers the options to []string, which is
// the format with which we include them into a OONI Measurement
func experimentOptionsToStringList(options map[string]any) (out []string) {
	for key, value := range options {
		out = append(out, fmt.Sprintf("%s=%v", key, value))
	}
	return
}
