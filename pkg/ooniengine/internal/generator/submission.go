package main

import "github.com/ooni/probe-cli/v3/internal/model"

//
// Measurement submission.
//

func init() {
	OONIEngine.Constants = append(OONIEngine.Constants, Constant{
		Docs: []string{
			"The error string inside SubmitEvent when the user disabled submission.",
		},
		Name:  "SubmissionDisabledError",
		Value: model.ErrSubmissionDisabled.Error(),
	})

	registerNewEvent(
		"Contains the results of a measurement submission.",
		"Submit",
		StructField{
			Docs: []string{
				"Failure that occurred or empty string (on success)",
			},
			Name: "Failure",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"Index of this measurement relative to the current experiment.",
			},
			Name: "Index",
			Type: TypeInt64,
		},
		StructField{
			Docs: []string{
				"The measurement's input.",
			},
			Name: "Input",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"The measurement's report ID.",
			},
			Name: "ReportID",
			Type: TypeString,
		},
		StructField{
			Docs: []string{
				"UTF-8 string containing serialized JSON measurement.",
			},
			Name: "Measurement",
			Type: TypeString,
		},
	)
}
