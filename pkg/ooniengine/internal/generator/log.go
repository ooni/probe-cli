package main

//
// Log definitions
//

// LogEventName is the name of the 'log' event.
var LogEventName = registerNewEvent(
	"A log message.",
	"Log",
	StructField{
		Docs: []string{
			"Log level.",
		},
		Name: "Level",
		Type: TypeString,
	},
	StructField{
		Docs: []string{
			"Log message.",
		},
		Name: "Message",
		Type: TypeString,
	},
)

func init() {
	// add constants
	consts := []Constant{{
		Docs: []string{
			"Debug log level.",
		},
		Name:  "LogLevelDebug",
		Value: "DEBUG",
	}, {
		Docs: []string{
			"Info log level.",
		},
		Name:  "LogLevelInfo",
		Value: "INFO",
	}, {
		Docs: []string{
			"Warning log level.",
		},
		Name:  "LogLevelWarning",
		Value: "WARNING",
	}}
	OONIEngine.Constants = append(OONIEngine.Constants, consts...)
}
