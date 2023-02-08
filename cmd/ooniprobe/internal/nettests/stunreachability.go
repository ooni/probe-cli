package nettests

import (
	"errors"
)

// STUNReachability nettest implementation.
type STUNReachability struct{}

// Run starts the nettest.
func (n STUNReachability) Run(ctl *Controller) error {
	return errors.New("not implemented")
}
