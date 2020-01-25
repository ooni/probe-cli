package nettests

// Tor test implementation
type Tor struct {
}

// Run starts the test
func (h Tor) Run(ctl *Controller) error {
	builder, err := ctl.Ctx.Session.NewExperimentBuilder(
		"tor",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// TorTestKeys contains the test keys
type TorTestKeys struct {
	IsAnomaly     bool    `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h Tor) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	testKeys := TorTestKeys{IsAnomaly: false}
	return testKeys, nil
}

// LogSummary writes the summary to the standard output
func (h Tor) LogSummary(s string) error {
	return nil
}
