package main

//
// Meta information about experiments
//

func init() {
	const experiment = "MetaInfoExperiment"

	config := registerNewConfig(
		"Config for the meta-info-experiment task.",
		experiment,
	)

	registerNewEvent(
		"Contains meta-info about an experiment",
		experiment,
		StructField{
			Docs: []string{
				"The experiment name",
			},
			Name: "Name",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Whether this experiment could use input.",
				"",
				"If this field is false, it does not make sense to generate",
				"command line options for passing input to the experiment.",
			},
			Name: "UsesInput",
			Type: TypeBool,
		},
	)

	// add tasks
	OONIEngine.Tasks = append(OONIEngine.Tasks, Task{
		Name:   experiment,
		Config: config,
	})
}
