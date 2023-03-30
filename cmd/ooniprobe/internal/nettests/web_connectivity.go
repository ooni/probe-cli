package nettests

import (
	"errors"
)

// WebConnectivity test implementation
type WebConnectivity struct{}

// Run starts the test
func (n WebConnectivity) Run(ctl *Controller) error {
	results, err := ctl.Session.CheckInResult()
	if err != nil {
		return err
	}
	if results.Tests.WebConnectivity == nil {
		return errors.New("no web_connectivity data")
	}
	urls, err := ctl.BuildAndSetInputIdxMap(results.Tests.WebConnectivity.URLs)
	if err != nil {
		return err
	}
	return ctl.Run(
		"web_connectivity",
		results.Tests.WebConnectivity.ReportID,
		urls,
	)
}
