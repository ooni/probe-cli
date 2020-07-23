package nettests

import (
	"errors"
)

// HTTPHeaderFieldManipulation test implementation
type HTTPHeaderFieldManipulation struct {
}

// Run starts the test
func (h HTTPHeaderFieldManipulation) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"http_header_field_manipulation",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// HTTPHeaderFieldManipulationTestKeys for the test
type HTTPHeaderFieldManipulationTestKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetTestKeys returns a projection of the tests keys needed for the views
func (h HTTPHeaderFieldManipulation) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	testKeys := HTTPHeaderFieldManipulationTestKeys{IsAnomaly: false}
	tampering, ok := tk["tampering"].(map[string]interface{})
	if !ok {
		return testKeys, errors.New("tampering testkey is invalid")
	}
	for _, v := range tampering {
		t, ok := v.(bool)
		// Ignore non booleans in the tampering map
		if ok && t == true {
			testKeys.IsAnomaly = true
		}
	}

	return testKeys, nil
}

// LogSummary writes the summary to the standard output
func (h HTTPHeaderFieldManipulation) LogSummary(s string) error {
	return nil
}
