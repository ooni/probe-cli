package main

//
// Data usage
//

func init() {
	registerNewEvent(
		"Information about the amount of data consumed by an experiment.",
		"DataUsage",
		StructField{
			Docs: []string{
				"KiB sent by this experiment.",
			},
			Name: "KibiBytesSent",
			Type: TypeFloat64,
		},
		StructField{
			Docs: []string{
				"KiB received by this experiment.",
			},
			Name: "KibiBytesReceived",
			Type: TypeFloat64,
		},
	)
}
