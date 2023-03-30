package nettests

// Psiphon test implementation
type Psiphon struct {
}

// Run starts the test
func (h Psiphon) Run(ctl *Controller) error {
	return ctl.Run(
		"psiphon",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
