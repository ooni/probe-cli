package nettests

// VanillaTor test implementation
type VanillaTor struct {
}

// Run starts the test
func (h VanillaTor) Run(ctl *Controller) error {
	return ctl.Run(
		"vanilla_tor",
		"", // TODO(bassosimone)
		[]string{""},
	)
}

func (h VanillaTor) onlyBackground() {}
