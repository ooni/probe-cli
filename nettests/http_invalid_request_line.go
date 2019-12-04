package nettests

import (
	"errors"
)

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *Controller) error {
	builder, err := ctl.Ctx.Session.NewExperimentBuilder(
		"http_invalid_request_line",
	)
	if err != nil {
		return err
	}
	if err := builder.SetOptionString("LogLevel", "INFO"); err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
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
