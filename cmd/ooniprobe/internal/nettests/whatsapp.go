package nettests

// WhatsApp test implementation
type WhatsApp struct {
}

// Run starts the test
func (h WhatsApp) Run(ctl *Controller) error {
	return ctl.Run(
		"whatsapp",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
