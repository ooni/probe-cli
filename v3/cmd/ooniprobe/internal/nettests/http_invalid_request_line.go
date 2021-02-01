package nettests

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"http_invalid_request_line",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}
