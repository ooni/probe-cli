package nettests

// ECHCheck nettest implementation.
type ECHCheck struct{}

// Run starts the nettest.
func (n ECHCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("echcheck")
	if err != nil {
		return err
	}
	// providing an input containing an empty string causes the experiment
	// to recognize the empty string and use the default URL
	return ctl.Run(builder, []string{""})
}
