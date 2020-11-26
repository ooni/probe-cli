package nettests

// NDT test implementation. We use v7 of NDT since 2020-03-12.
type NDT struct {
}

// Run starts the test
func (n NDT) Run(ctl *Controller) error {
	// Since 2020-03-18 probe-engine exports v7 as "ndt".
	builder, err := ctl.Session.NewExperimentBuilder("ndt")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
