package nettests

// WhatsApp test implementation
type WhatsApp struct {
}

// Run starts the test
func (h WhatsApp) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"whatsapp",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
