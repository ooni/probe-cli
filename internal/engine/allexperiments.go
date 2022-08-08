package engine

//
// List of all implemented experiments.
//

import "github.com/ooni/probe-cli/v3/internal/registry"

// AllExperiments returns the name of all experiments
func AllExperiments() []string {
	return registry.ExperimentNames()
}
