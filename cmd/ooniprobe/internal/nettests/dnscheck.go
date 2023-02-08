package nettests

import (
	"errors"
)

// DNSCheck nettest implementation.
type DNSCheck struct{}

// Run starts the nettest.
func (n DNSCheck) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
