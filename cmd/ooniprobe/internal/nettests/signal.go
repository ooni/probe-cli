package nettests

// Signal test implementation
type Signal struct {
}

// Run starts the test
func (h Signal) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"signal",
	)
	if err != nil {
		return err
	}

	return ctl.Run(builder, []string{""})
}
