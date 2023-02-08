package nettests

import "errors"

// Tor test implementation
type Tor struct {
}

// Run starts the test
func (h Tor) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
