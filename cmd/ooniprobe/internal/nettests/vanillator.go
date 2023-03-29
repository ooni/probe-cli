package nettests

import "errors"

// VanillaTor test implementation
type VanillaTor struct {
}

// Run starts the test
func (h VanillaTor) Run(ctl *Controller) error {
	return errors.New("not implemented")
}

func (h VanillaTor) onlyBackground() {}
