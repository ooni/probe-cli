package nettests

// Signal nettest implementation.
type Signal struct{}

// Run starts the nettest.
func (h Signal) Run(ctl *Controller) error {
	return ctl.Run(
		"signal",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
