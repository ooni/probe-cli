package middlebox

import (
	"errors"

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
