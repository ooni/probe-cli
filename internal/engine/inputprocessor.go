package engine

import (
	"context"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// InputProcessorExperiment is the Experiment
// according to InputProcessor.
type InputProcessorExperiment interface {
	MeasureAsync(
		ctx context.Context, input string) (<-chan *model.Measurement, error)
}

// InputProcessor processes inputs. We perform a Measurement
// for each input using the given Experiment.
type InputProcessor struct {
	// Annotations contains the measurement annotations
	Annotations map[string]string

	// Experiment is the code that will run the experiment.
	Experiment InputProcessorExperiment

	// Inputs is the list of inputs to measure.
	Inputs []model.OOAPIURLInfo

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MaxRuntime is the optional maximum runtime
	// when looping over a list of inputs (e.g. when
	// running Web Connectivity). Zero means that
	// there will be no MaxRuntime limit.
	MaxRuntime time.Duration

	// Options contains command line options for this experiment.
	Options []string

	// Parallelism contains the OPTIONAL parallelism
	// for performing measurements. A zero or negative
	// value implies we want just one goroutine.
	Parallelism int

	// Saver is the code that will save measurement results
	// on persistent storage (e.g. the file system).
	Saver Saver

	// Submitter is the code that will submit measurements
	// to the OONI collector.
	Submitter Submitter
}

// Run processes all the input subject to the duration of the
// context. The code will perform measurements using the given
// experiment; submit measurements using the given submitter;
// save measurements using the given saver.
//
// Annotations and Options will be saved in the measurement.
//
// The default behaviour of this code is that an error while
// measuring, while submitting, or while saving a measurement
// is always causing us to break out of the loop. The user
// though is free to choose different policies by configuring
// the Experiment, Submitter, and Saver fields properly.
func (ip *InputProcessor) Run(ctx context.Context) error {
	// TODO(bassosimone): it's unclear how to report errors back
	// now that we're in a parallel context.
	ip.run(ctx)
	return nil
}

// These are the reasons why run could stop.
const (
	stopNormal = (1 << iota)
	stopMaxRuntime
)

// run is like Run but, in addition to returning an error, it
// also returns the reason why we stopped.
func (ip *InputProcessor) run(ctx context.Context) {
	saver := StartAsyncSaver(ip.Saver)
	submitter := StartAsyncSubmitter(ip.Logger, ip.Submitter, saver)
	wg := &sync.WaitGroup{}
	urls := ip.generateInputs()
	parallelism := ip.Parallelism
	if parallelism < 1 {
		parallelism = 1
	}
	start := time.Now()
	for cnt := 0; cnt < parallelism; cnt++ {
		wg.Add(1)
		go ip.performMeasurement(ctx, wg, urls, start, submitter)
	}
	// wait for measurers to join
	wg.Wait()
	// termination protocol for saver and submitter
	submitter.Stop()
	submitter.Wait()

	saver.Stop()
	saver.Wait()
}

func (ip *InputProcessor) performMeasurement(
	ctx context.Context, wg *sync.WaitGroup, urls <-chan *inputWithIndex,
	start time.Time, submitter AsyncSubmitter) (int, error) {
	defer wg.Done() // synchronize with the parent
	for inputIdx := range urls {
		idx := inputIdx.idx
		input := inputIdx.input
		if ip.MaxRuntime > 0 && time.Since(start) > ip.MaxRuntime {
			return stopMaxRuntime, nil
		}
		if input != "" {
			ip.Logger.Infof("[%d/%d] running with input: %s", idx+1, len(ip.Inputs), input)
		}
		source, err := ip.Experiment.MeasureAsync(ctx, input)
		if err != nil {
			return 0, err
		}
		for meas := range source {
			submitter.Submit(idx, meas)
		}
	}
	return stopNormal, nil
}

// inputWithIndex combines an input with its index.
type inputWithIndex struct {
	// idx is the index
	idx int

	// input contains the URL input
	input string
}

// generateInputs returns a channel where each input is emitted.
func (ip *InputProcessor) generateInputs() <-chan *inputWithIndex {
	out := make(chan *inputWithIndex)
	go func() {
		defer close(out)
		for idx, url := range ip.Inputs {
			out <- &inputWithIndex{
				idx:   idx,
				input: url.URL,
			}
		}
	}()
	return out
}
