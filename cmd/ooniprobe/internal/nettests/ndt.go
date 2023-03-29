package nettests

import "errors"

// NDT test implementation. We use v7 of NDT since 2020-03-12.
type NDT struct {
}

// Run starts the test
func (n NDT) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
