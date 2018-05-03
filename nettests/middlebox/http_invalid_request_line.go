package middlebox

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
)

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *nettests.Controller) error {
	mknt := mk.NewNettest("HttpInvalidRequestLine")
	ctl.Init(mknt)
	return mknt.Run()
}

// HTTPInvalidRequestLineSummary for the test
type HTTPInvalidRequestLineSummary struct {
	Tampering bool
}

// Summary generates a summary for a test run
func (h HTTPInvalidRequestLine) Summary(tk map[string]interface{}) interface{} {
	tampering := tk["tampering"].(bool)

	return HTTPInvalidRequestLineSummary{
		Tampering: tampering,
	}
}

// LogSummary writes the summary to the standard output
func (h HTTPInvalidRequestLine) LogSummary(s string) error {
	return nil
}
