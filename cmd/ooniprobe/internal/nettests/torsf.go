package nettests

// TorSf test implementation
type TorSf struct {
}

// Run starts the test
func (h TorSf) Run(ctl *Controller) error {
	return ctl.Run(
		"torsf",
		"", // TODO(bassosimone)
		[]string{""},
	)
}

func (h TorSf) onlyBackground() {}
