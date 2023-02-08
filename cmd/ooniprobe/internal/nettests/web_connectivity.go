package nettests

import (
	"github.com/ooni/probe-cli/v3/internal/nettests"
)

func (n WebConnectivity) lookupURLs(
	ctl *Controller,
	factory *nettests.WebConnectivityFactory,
) ([]string, error) {
	testlist, err := factory.LoadInputs()
	if err != nil {
		return nil, err
	}
	return ctl.BuildAndSetInputIdxMap(testlist)
}

// WebConnectivity test implementation
type WebConnectivity struct{}

// Run starts the test
func (n WebConnectivity) Run(ctl *Controller) error {
	factoryConfig := &nettests.WebConnectivityFactoryConfig{
		CheckIn:    ctl.CheckInResult,
		InputFiles: ctl.InputFiles,
		Inputs:     ctl.Inputs,
		Session:    ctl.Session,
	}
	factory, err := nettests.NewWebConnectivityFactory(factoryConfig)
	if err != nil {
		return err
	}
	urls, err := n.lookupURLs(ctl, factory)
	if err != nil {
		return err
	}
	return ctl.Run(factory, urls)
}
