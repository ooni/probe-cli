package nettests

import "errors"

// HTTPInvalidRequestLine test implementation
type HTTPInvalidRequestLine struct {
}

// Run starts the test
func (h HTTPInvalidRequestLine) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
