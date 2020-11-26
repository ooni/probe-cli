package nettests

// HTTPHeaderFieldManipulation test implementation
type HTTPHeaderFieldManipulation struct {
}

// Run starts the test
func (h HTTPHeaderFieldManipulation) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"http_header_field_manipulation",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
