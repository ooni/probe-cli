package nettests

// Dash test implementation
type Dash struct {
}

// Run starts the test
func (d Dash) Run(ctl *Controller) error {
	return ctl.Run(
		"dash",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
