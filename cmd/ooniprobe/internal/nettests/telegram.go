package nettests

// Telegram test implementation
type Telegram struct {
}

// Run starts the test
func (h Telegram) Run(ctl *Controller) error {
	return ctl.Run(
		"telegram",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
