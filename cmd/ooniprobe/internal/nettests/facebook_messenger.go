package nettests

// FacebookMessenger test implementation
type FacebookMessenger struct {
}

// Run starts the test
func (h FacebookMessenger) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"facebook_messenger",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
