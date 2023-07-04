package nettests

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *Controller) error {
	return ctl.Run(
		"http_invalid_request_line",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
