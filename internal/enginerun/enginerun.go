// Package enginerun implements running a single nettest.
package enginerun

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Nettest describes a nettest to run. Make sure you fill MANDATORY fields.
type Nettest struct {
	// Inputs contains MANDATORY inputs for the nettest. If the nettest does not take any input,
	// you MUST fill this value using a single entry containing an empty string.
	Inputs []string `json:"inputs"`

	// Options contains the nettest options. Any option name starting with
	// `Safe` will be available for the nettest run, but omitted from
	// the serialized Measurement when we submit it to the OONI backend.
	Options map[string]any `json:"options"`

	// TestName contains the MANDATORY nettest name.
	TestName string `json:"test_name"`
}

// Session is the measurement session.
//
// The engine.Session type implements this interface.
type Session interface {
	// Logger returns the logger to use.
	Logger() model.Logger

	// NewExperimentBuilder creates a new model.ExperimentBuilder.
	NewExperimentBuilder(name string) (model.ExperimentBuilder, error)
}

// config contains configuration for [Run].
type config struct {
	// parallelism defines the number of goroutines that
	// should run in parallel and measure.
	parallelism int
}

// Option is an option for [Run].
type Option func(cfg *config)

// OptionParallelism configures the number of parallel goroutines
// that should perform concurrent measurements.
//
// Setting a value <= 1 is equivalent to setting 1 as the value.
//
// The default value of this option is 1.
func OptionParallelism(value int) Option {
	return func(cfg *config) {
		switch {
		case value > 1:
			cfg.parallelism = value
		default:
			cfg.parallelism = 1
		}
	}
}

// RunError is the event emitted when we cannot run a nettest.
type RunError struct {
	// Err is the error that occurred.
	Err error

	// Index is the input index.
	Index int

	// Input is the input value.
	Input string
}

// RunSuccess is the event emitted after we successfully ran a nettest.
type RunSuccess struct {
	// Index is the input index.
	Index int

	// Input is the input value.
	Input string

	// Measurement is the measurement.
	Measurement *model.Measurement
}

// DataUsage contains information about the data consumed by running a nettest.
type DataUsage struct {
	KibiBytesReceived float64
	KibiBytesSent     float64
}

// Events allows to access the channels where the goroutines created by [Start] emit events.
type Events struct {
	dataUsage  chan *DataUsage
	done       chan any
	runError   chan *RunError
	runSuccess chan *RunSuccess
}

// DataUsage returns the channel where we return overall data usage information.
func (ev *Events) DataUsage() <-chan *DataUsage {
	return ev.dataUsage
}

// Done returns the channel closed when done measuring.
func (ev *Events) Done() <-chan any {
	return ev.done
}

// RunError returns the channel where we post cases where a measurement failed.
func (ev *Events) RunError() <-chan *RunError {
	return ev.runError
}

// RunSuccess returns the channel where we post successful measurements.
func (ev *Events) RunSuccess() <-chan *RunSuccess {
	return ev.runSuccess
}

// inputIdx contains input and its index.
type inputIdx struct {
	idx   int
	input string
}

// Start starts running the given [Nettest] using the given options using background
// goroutines. This function returns an error if it cannot create the nettest. If the
// error is nil, the returned struct contains channels where we emit events.
func Start(ctx context.Context, sess Session, nt *Nettest, options ...Option) (*Events, error) {
	// 1. create experiment builder
	builder, err := sess.NewExperimentBuilder(nt.TestName)
	if err != nil {
		return nil, err
	}

	// 2. configure experiment options
	if err := builder.SetOptionsAny(nt.Options); err != nil {
		return nil, err
	}

	// 3. construct the experiment instance
	experiment := builder.NewExperiment()

	// 4. create a generator that produces input
	inputs := produce(nt)

	// 5. initialize the options
	cfg := &config{
		parallelism: 1,
	}
	for _, opt := range options {
		opt(cfg)
	}

	// 6. create the output structure
	events := &Events{
		dataUsage:  make(chan *DataUsage),
		done:       make(chan any),
		runError:   make(chan *RunError),
		runSuccess: make(chan *RunSuccess),
	}

	// 7. start the required number of runners
	wg := &sync.WaitGroup{}
	for idx := 0; idx <= cfg.parallelism; idx++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			consume(ctx, experiment, inputs, events)
		}()
	}

	// 8. make sure we close events when done
	go func() {
		defer close(events.done)
		wg.Wait()
		events.dataUsage <- &DataUsage{
			KibiBytesReceived: experiment.KibiBytesReceived(),
			KibiBytesSent:     experiment.KibiBytesSent(),
		}
	}()

	// 9. return to the caller
	return events, nil
}

// produce generates a stream of the inputs along with their index.
func produce(nt *Nettest) <-chan *inputIdx {
	inputs := make(chan *inputIdx)
	go func() {
		defer close(inputs)
		for idx, input := range nt.Inputs {
			inputs <- &inputIdx{
				idx:   idx,
				input: input,
			}
		}
	}()
	return inputs
}

// consume transforms inputs into events.
func consume(ctx context.Context, experiment model.Experiment, inputs <-chan *inputIdx, events *Events) {
	for input := range inputs {
		// TODO(bassosimone): are experiments safe to run concurrently? Maybe
		// we should double check this optimistic assumption!
		meas, err := experiment.MeasureWithContext(ctx, input.input)

		if err != nil {
			events.runError <- &RunError{
				Err:   err,
				Index: input.idx,
				Input: input.input,
			}
			continue
		}

		events.runSuccess <- &RunSuccess{
			Index:       input.idx,
			Input:       input.input,
			Measurement: meas,
		}
	}
}
