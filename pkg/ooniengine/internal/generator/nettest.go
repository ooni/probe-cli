package main

//
// Nettest task
//

func init() {
	const basename = "Nettest"

	config := registerNewConfig(
		"Config for running a nettest.",
		basename,
		StructField{
			Docs: []string{
				"OPTIONAL annotations for the nettest.",
			},
			Name: "Annotations",
			Type: TypeMapStringString,
		},
		StructField{
			Docs: []string{
				"OPTIONAL extra options for the nettest.",
			},
			Name: "ExtraOptions",
			Type: TypeMapStringAny,
		},
		StructField{
			Docs: []string{
				"OPTIONAL inputs for the nettest.",
			},
			Name: "Inputs",
			Type: TypeListString,
		},
		StructField{
			Docs: []string{
				"An OPTIONAL list of files from which to read inputs for the nettest.",
			},
			Name: "InputFilePaths",
			Type: TypeListString,
		},
		StructField{
			Docs: []string{
				"The OPTIONAL nettest maximum runtime in seconds.",
				"",
				"This setting only applies to nettests that require input, such",
				"as Web Connectivity.",
			},
			Name: "MaxRuntime",
			Type: TypeInt64,
		},
		StructField{
			Docs: []string{
				"The MANDATORY name of the nettest to execute.",
			},
			Name: "Name",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"This setting allows to OPTIONALLY disable submitting measurements.",
				"",
				"The default is that we submit every measurement we perform.",
			},
			Name: "NoCollector",
			Type: TypeBool,
		},
		StructField{
			Docs: []string{
				"This setting allows to OPTIONALLY disable saving measurements to disk.",
				"",
				"The default is to save using the file name indicated by ReportFile.",
			},
			Name: "NoJSON",
			Type: TypeBool,
		},
		StructField{
			Docs: []string{
				"OPTIONALLY tells the engine to randomly shuffle the input list.",
			},
			Name: "Random",
			Type: TypeBool,
		},
		StructField{
			Docs: []string{
				"The OPTIONAL name of the file where to save measurements.",
				"",
				"If this field is empty, we will use 'report.jsonl' as the file name.",
			},
			Name: "ReportFile",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Config for creating a session.",
			},
			Name: "Session",
			Type: "SessionConfig",
		},
	)

	// add tasks
	OONIEngine.Tasks = append(OONIEngine.Tasks, Task{
		Name:   basename,
		Config: config,
	})
}
