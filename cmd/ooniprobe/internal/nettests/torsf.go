package nettests

// TorSf test implementation
type TorSf struct {
}

// Run starts the test
func (h TorSf) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("torsf")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

func (h TorSf) onlyBackground() {}
