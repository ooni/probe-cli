package nettests

// HTTPHeaderFieldManipulation test implementation
type HTTPHeaderFieldManipulation struct {
}

// Run starts the test
func (h HTTPHeaderFieldManipulation) Run(ctl *Controller) error {
	return ctl.Run(
		"http_header_field_manipulation",
		"", // TODO(bassosimone)
		[]string{""},
	)
}
