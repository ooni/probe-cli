package nettests

// FacebookMessenger test implementation
type FacebookMessenger struct {
}

// Run starts the test
func (h FacebookMessenger) Run(ctl *Controller) error {
	return ctl.Run(
		"facebook_messenger",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
