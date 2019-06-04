package middlebox

import (
	"errors"

	"github.com/ooni/probe-cli/nettests"
	"github.com/ooni/probe-engine/experiment/hirl"
)

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *nettests.Controller) error {
	experiment := hirl.NewExperiment(ctl.Ctx.Session, hirl.Config{
		LogLevel: "INFO",
	})
	return ctl.Run(experiment, []string{""})
}

// HTTPInvalidRequestLineTestKeys for the test
type HTTPInvalidRequestLineTestKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h HTTPInvalidRequestLine) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	testKeys := HTTPInvalidRequestLineTestKeys{IsAnomaly: false}

	tampering, ok := tk["tampering"].(bool)
	if !ok {
		return testKeys, errors.New("tampering is not bool")
	}
	testKeys.IsAnomaly = tampering

	return testKeys, nil
}

// LogSummary writes the summary to the standard output
func (h HTTPInvalidRequestLine) LogSummary(s string) error {
	return nil
}
