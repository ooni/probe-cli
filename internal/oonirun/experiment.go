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
// are marked as MANDATORY, or the Experiment would cause crashes.
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
}

// Run runs the given experiment.
func (ed *Experiment) Run(ctx context.Context) error {

	// 1. create experiment builder
	builder, err := ed.Session.NewExperimentBuilder(ed.Name)
	if err != nil {
		return err
	}

	// 2. create input loader and load input for this experiment
	inputLoader := &engine.InputLoader{
		CheckInConfig: &model.OOAPICheckInConfig{
			RunType:  model.RunTypeManual,
			OnWiFi:   true, // meaning: not on 4G
			Charging: true,
		},
		ExperimentName: ed.Name,
		InputPolicy:    builder.InputPolicy(),
		StaticInputs:   ed.Inputs,
		SourceFiles:    ed.InputFilePaths,
		Session:        ed.Session,
	}
	inputs, err := inputLoader.Load(ctx)
	if err != nil {
		return err
	}

	// 3. randomize input, if needed
	if ed.Random {
		rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
		rnd.Shuffle(len(inputs), func(i, j int) {
			inputs[i], inputs[j] = inputs[j], inputs[i]
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
	submitter, err := engine.NewSubmitter(ctx, engine.SubmitterConfig{
		Enabled: !ed.NoCollector,
		Session: ed.Session,
		Logger:  ed.Session.Logger(),
	})
	if err != nil {
		return err
	}

	// 7. create the saver
	saver, err := engine.NewSaver(engine.SaverConfig{
		Enabled:    !ed.NoJSON,
		Experiment: experiment,
		FilePath:   ed.ReportFile,
		Logger:     logger,
	})
	if err != nil {
		return err
	}

	// 8. create an input processor
	inputProcessor := &engine.InputProcessor{
		Annotations: ed.Annotations,
		Experiment: &experimentWrapper{
			child:  engine.NewInputProcessorExperimentWrapper(experiment),
			logger: ed.Session.Logger(),
			total:  len(inputs),
		},
		Inputs:     inputs,
		MaxRuntime: time.Duration(ed.MaxRuntime) * time.Second,
		Options:    experimentOptionsToStringList(ed.ExtraOptions),
		Saver:      engine.NewInputProcessorSaverWrapper(saver),
		Submitter: &exprimentSubmitterWrapper{
			child:  engine.NewInputProcessorSubmitterWrapper(submitter),
			logger: ed.Session.Logger(),
		},
	}

	// 9. process input and generate measurements
	return inputProcessor.Run(ctx)
}

// experimentOptionsToStringList convers the options to []string, which is
// the format with which we include them into a OONI Measurement
func experimentOptionsToStringList(options map[string]any) (out []string) {
	for key, value := range options {
		out = append(out, fmt.Sprintf("%s=%v", key, value))
	}
	return
}

// experimentWrapper wraps an experiment and logs progress
type experimentWrapper struct {
	// child is the child experiment wrapper
	child engine.InputProcessorExperimentWrapper

	// logger is the logger to use
	logger model.Logger

	// total is the total number of inputs
	total int
}

func (ew *experimentWrapper) MeasureAsync(
	ctx context.Context, input string, idx int) (<-chan *model.Measurement, error) {
	if input != "" {
		ew.logger.Infof("[%d/%d] running with input: %s", idx+1, ew.total, input)
	}
	return ew.child.MeasureAsync(ctx, input, idx)
}

// exprimentSubmitterWrapper implements a submission policy where we don't
// fail if we cannot submit a measurement
type exprimentSubmitterWrapper struct {
	// child is the child submitter wrapper
	child engine.InputProcessorSubmitterWrapper

	// logger is the logger to use
	logger model.Logger
}

func (sw *exprimentSubmitterWrapper) Submit(ctx context.Context, idx int, m *model.Measurement) error {
	if err := sw.child.Submit(ctx, idx, m); err != nil {
		sw.logger.Warnf("submitting measurement failed: %s", err.Error())
	}
	// policy: we do not stop the loop if measurement submission fails
	return nil
}
