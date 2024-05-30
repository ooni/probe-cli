package registry

import "sort"

// Where we register all the available experiments.
var AllExperiments = map[string]func() *Factory{}

// ExperimentNames returns the name of all experiments
func ExperimentNames() (names []string) {
	for key := range AllExperiments {
		names = append(names, key)
	}
	sort.Strings(names) // sort by name to always provide predictable output
	return
}
