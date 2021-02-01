package nettests

// Telegram test implementation
type Telegram struct {
}

// Run starts the test
func (h Telegram) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"telegram",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
