package nettests

import "errors"

// TorSf test implementation
type TorSf struct {
}

// Run starts the test
func (h TorSf) Run(ctl *Controller) error {
	return errors.New("not implemented")
}

func (h TorSf) onlyBackground() {}
