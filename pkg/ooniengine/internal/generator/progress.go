package main

func init() {
	registerNewEvent(
		"Provides information about nettests' progress.",
		"Progress",
		StructField{
			Docs: []string{
				"Number between 0 and 1 indicating the current progress.",
			},
			Name: "Percentage",
			Type: TypeFloat64,
		},
		StructField{
			Docs: []string{
				"Message associated with the current progress indication.",
			},
			Name: "Message",
			Type: TypeString,
		},
	)
}
