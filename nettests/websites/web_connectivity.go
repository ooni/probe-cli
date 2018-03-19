package websites

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/openobservatory/gooni/nettests"
)

// WebConnectivity test implementation
type WebConnectivity struct {
}

// Run starts the test
func (n WebConnectivity) Run(ctl *nettests.Controller) error {
	nt := mk.NewNettest("WebConnectivity")
	ctl.Init(nt)
	return nt.Run()
}

// Summary generates a summary for a test run
func (n WebConnectivity) Summary(tk map[string]interface{}) interface{} {
	return nil
}

// LogSummary writes the summary to the standard output
func (n WebConnectivity) LogSummary(s string) error {
	return nil
}
