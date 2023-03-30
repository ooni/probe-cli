package nettests

// Tor test implementation
type Tor struct {
}

// Run starts the test
func (h Tor) Run(ctl *Controller) error {
	return ctl.Run(
		"tor",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
