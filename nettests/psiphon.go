package nettests

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
	IsAnomaly bool `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h Psiphon) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	return PsiphonTestKeys{
		IsAnomaly: tk["failure"] != nil,
	}, nil
}

// LogSummary writes the summary to the standard output
func (h Psiphon) LogSummary(s string) error {
	return nil
}
