package registryx

import "github.com/ooni/probe-cli/v3/internal/model"

// Where we register all the available experiments.
var AllExperiments = map[string]Experiment{}

// ExperimentNames returns the name of all experiments
func ExperimentNames() (names []string) {
	for key := range AllExperiments {
		names = append(names, key)
	}
	return
}

var AllExperimentOptions = map[string]model.ExperimentOptions{}
