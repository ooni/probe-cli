package engine

//
// List of all implemented experiments.
//
// Note: if you're looking for a way to register a new experiment, we
// now use the internal/registry package for this purpose.
//
// (This comment will eventually autodestruct.)
//

import "github.com/ooni/probe-cli/v3/internal/registry"

// AllExperiments returns the name of all experiments
func AllExperiments() []string {
	return registry.ExperimentNames()
}
