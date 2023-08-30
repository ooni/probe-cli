package nettests

// ECHCheck nettest implementation.
type ECHCheck struct{}

// Run starts the nettest.
func (n ECHCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("echcheck")
	if err != nil {
		return err
	}
	// providing empty input prompts the experiment to use the default URL https://example.org
	return ctl.Run(builder, []string{})
}
