package registry

// Where we register all the available experiments.
var AllExperiments = map[string]*Factory{}

// ExperimentNames returns the name of all experiments
func ExperimentNames() (names []string) {
	for key := range AllExperiments {
		names = append(names, key)
	}
	return
}
