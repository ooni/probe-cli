package nettests

// RiseupVPN test implementation
type RiseupVPN struct {
}

// Run starts the test
func (h RiseupVPN) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"riseupvpn",
	)
	if err != nil {
		return err
	}

	return ctl.Run(builder, []string{""})
}
