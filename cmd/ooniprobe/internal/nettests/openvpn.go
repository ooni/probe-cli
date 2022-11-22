package nettests

// OpenVPN test implementation
type OpenVPN struct {
}

// Run starts the test
func (h OpenVPN) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("openvpn")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
