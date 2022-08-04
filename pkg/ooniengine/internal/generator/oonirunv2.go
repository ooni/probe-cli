package main

import "github.com/ooni/probe-cli/v3/internal/runtimex"

//
// OONI Run v2
//

// OONIRunV2NettestListType is the name of a list of OONI Run V2Nettest
// entries, which needs special handling when compiling to Go, Dart.
const OONIRunV2NettestListType = "::OONIRunV2NettestList"

func init() {
	nettestStruct := registerNewStruct(
		"OONI Run v2 nettest descriptor.",
		"OONIRunV2Nettest",
		StructField{
			Docs: []string{
				"OPTIONAL annotations for the nettest.",
			},
			Name: "Annotations",
			Type: TypeMapStringString,
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
				"OPTIONAL extra options for the nettest.",
			},
			Name: "Options",
			Type: TypeMapStringAny,
		},
		StructField{
			Docs: []string{
				"The MANDATORY name of the nettest to execute.",
			},
			Name: "TestName",
			Type: TypeString,
		},
	)

	runtimex.PanicIfFalse(
		("::"+nettestStruct+"List") == OONIRunV2NettestListType,
		"oonirunv2.go: mismatch between OONIRunV2NettestListType and nettestStruct",
	)

	descriptorStruct := registerNewStruct(
		"OONI Run v2 descriptor.",
		"OONIRunV2Descriptor",
		StructField{
			Docs: []string{
				"Name of this OONI Run v2 descriptor.",
			},
			Name: "Name",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Description for this OONI Run v2 descriptor.",
			},
			Name: "Description",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Author of this OONI Run v2 descriptor.",
			},
			Name: "Author",
			Type: TypeString,
		},
		StructField{
			Docs: []string{},
			Name: "Nettests",
			Type: OONIRunV2NettestListType, // special handling
		},
	)

	measureDescriptor := "OONIRunV2MeasureDescriptor"

	measureDescriptorConfig := registerNewConfig(
		"Configures the OONI Run v2 task measuring an already available descriptor.",
		measureDescriptor,
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
		StructField{
			Docs: []string{
				"Descriptor for OONI Run v2",
			},
			Name: "Descriptor",
			Type: Type(descriptorStruct),
		},
	)

	OONIEngine.Tasks = append(OONIEngine.Tasks, Task{
		Name:   measureDescriptor,
		Config: measureDescriptorConfig,
	})
}
