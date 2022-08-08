package registry

// Where we register all the available experiments.
var allexperiments = map[string]*Factory{}

// ExperimentNames returns the name of all experiments
func ExperimentNames() (names []string) {
	for key := range allexperiments {
		names = append(names, key)
	}
	return
}
