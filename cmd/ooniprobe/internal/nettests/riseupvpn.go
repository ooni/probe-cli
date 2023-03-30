package nettests

// RiseupVPN test implementation
type RiseupVPN struct {
}

// Run starts the test
func (h RiseupVPN) Run(ctl *Controller) error {
	return ctl.Run(
		"riseupvpn",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
