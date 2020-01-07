package nettests

import "github.com/pkg/errors"

// Psiphon test implementation
type Psiphon struct {
}

// Run starts the test
func (h Psiphon) Run(ctl *Controller) error {
	builder, err := ctl.Ctx.Session.NewExperimentBuilder(
		"psiphon",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// PsiphonTestKeys contains the test keys
type PsiphonTestKeys struct {
	IsAnomaly     bool    `json:"-"`
	BootstrapTime float64 `json:"bootstrap_time"`
	Failure       string  `json:"failure"`
}

// GetTestKeys generates a summary for a test run
func (h Psiphon) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var (
		err error
		ok  bool
	)
	testKeys := PsiphonTestKeys{IsAnomaly: false, Failure: ""}
	if tk["failure"] != nil {
		testKeys.IsAnomaly = true
		testKeys.Failure, ok = tk["failure"].(string)
		if !ok {
			err = errors.Wrap(err, "failure key invalid")
		}
	}
	testKeys.BootstrapTime, ok = tk["bootstrap_time"].(float64)
	if !ok {
		err = errors.Wrap(err, "bootstrap_time key invalid")
	}
	return testKeys, err
}

// LogSummary writes the summary to the standard output
func (h Psiphon) LogSummary(s string) error {
	return nil
}
