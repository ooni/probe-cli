package nettests

import "errors"

// Psiphon test implementation
type Psiphon struct {
}

// Run starts the test
func (h Psiphon) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
