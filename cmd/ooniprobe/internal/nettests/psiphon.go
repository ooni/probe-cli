package nettests

// Psiphon test implementation
type Psiphon struct {
}

// Run starts the test
func (h Psiphon) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"psiphon",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
