package websites

import (
	"github.com/openobservatory/gooni/internal/database"
	"github.com/openobservatory/gooni/nettests"
)

// WebConnectivity test implementation
type WebConnectivity struct {
}

// Run starts the test
func (n WebConnectivity) Run(ctl *nettests.Controller) error {
	return nil
}

// Summary generates a summary for a test run
func (n WebConnectivity) Summary(m *database.Measurement) string {
	return ""
}

// LogSummary writes the summary to the standard output
func (n WebConnectivity) LogSummary(s string) error {
	return nil
}
