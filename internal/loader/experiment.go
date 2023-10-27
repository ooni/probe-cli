package loader

// ExperimentTarget is a target that an experiment should measure.
type ExperimentTarget struct {
	// Options contains the experiment options.
	Options map[string]any `json:"options"`

	// Inputs contains the experiment input.
	Input string `json:"input"`

	// CategoryCode contains the category code for this target.
	CategoryCode string `json:"category_code"`

	// CountryCode contains the country code for this target.
	CountryCode string `json:"country_code"`
}

// ExperimentSpec specifies and experiment to run.
type ExperimentSpec struct {
	// Name is the experiment canonical name.
	Name string `json:"name"`

	// Targets contains the experiment targets to measure.
	Targets []ExperimentTarget `json:"targets"`
}
