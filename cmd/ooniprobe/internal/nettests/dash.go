package nettests

import "errors"

// Dash test implementation
type Dash struct {
}

// Run starts the test
func (d Dash) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
