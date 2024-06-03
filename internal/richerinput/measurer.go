package richerinput

//
// Definition of the Measurer and related types
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// MeasurerRunArgs contains the arguments for the [Measurer] Run method.
type MeasurerRunArgs[T Target] struct {
	// Callbacks contains MANDATORY experiment callbacks.
	Callbacks model.RicherInputCallbacks

	// Measurement is the MANDATORY measurement in which the experiment
	// must write the results of the measurement.
	Measurement *model.Measurement

	// Session is the MANDATORY session the experiment can use.
	Session model.RicherInputSession

	// Target is MANDATORY and contains richer input.
	Target T
}

// Measurer measures using richer input.
type Measurer[T Target] interface {
	// ExperimentName returns the experiment name.
	ExperimentName() string

	// ExperimentVersion returns the experiment version.
	ExperimentVersion() string

	// Run runs the experiment with the specified context, session,
	// measurement, and experiment calbacks. This method should only
	// return an error in case the experiment could not run (e.g.,
	// a required input is missing). Otherwise, the code should just
	// set the relevant OONI error inside of the measurement and
	// return nil. This is important because the caller WILL NOT submit
	// the measurement if this method returns an error.
	Run(ctx context.Context, args *MeasurerRunArgs[T]) error
}

// VoidMeasurer is the measurer without input.
type VoidMeasurer struct {
	// Measurer is the MANDATORY model.Measurer to use.
	Measurer model.ExperimentMeasurer
}

var _ Measurer[VoidTarget] = &VoidMeasurer{}

// ExperimentName implements Measurer.
func (vm *VoidMeasurer) ExperimentName() string {
	return vm.Measurer.ExperimentName()
}

// ExperimentVersion implements Measurer.
func (vm *VoidMeasurer) ExperimentVersion() string {
	return vm.Measurer.ExperimentVersion()
}

// Run implements Measurer.
func (vm *VoidMeasurer) Run(ctx context.Context, args *MeasurerRunArgs[VoidTarget]) error {
	return vm.Measurer.Run(ctx, &model.ExperimentArgs{
		Callbacks:   args.Callbacks,
		Measurement: args.Measurement,
		Session:     args.Session,
	})
}
