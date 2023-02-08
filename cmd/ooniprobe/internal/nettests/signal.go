package nettests

import "errors"

// Signal nettest implementation.
type Signal struct{}

// Run starts the nettest.
func (h Signal) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
