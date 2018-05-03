package middlebox

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
)

// HTTPHeaderFieldManipulation test implementation
type HTTPHeaderFieldManipulation struct {
}

// Run starts the test
func (h HTTPHeaderFieldManipulation) Run(ctl *nettests.Controller) error {
	mknt := mk.NewNettest("HttpHeaderFieldManipulation")
	ctl.Init(mknt)
	return mknt.Run()
}

// HTTPHeaderFieldManipulationSummary for the test
type HTTPHeaderFieldManipulationSummary struct {
	Tampering bool
}

// Summary generates a summary for a test run
func (h HTTPHeaderFieldManipulation) Summary(tk map[string]interface{}) interface{} {
	tampering := false
	for _, v := range tk["tampering"].(map[string]interface{}) {
		t, ok := v.(bool)
		// Ignore non booleans in the tampering map
		if ok && t == true {
			tampering = true
		}
	}

	return HTTPHeaderFieldManipulationSummary{
		Tampering: tampering,
	}
}

// LogSummary writes the summary to the standard output
func (h HTTPHeaderFieldManipulation) LogSummary(s string) error {
	return nil
}
