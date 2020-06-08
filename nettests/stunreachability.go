package nettests

import "errors"

// STUNReachability test implementation.
type STUNReachability struct {
}

// Run starts the test
func (n STUNReachability) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("stun_reachability")
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// STUNReachabilityTestKeys for the test
type STUNReachabilityTestKeys struct {
	Failure   string `json:"failure"`
	IsAnomaly bool   `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (n STUNReachability) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var testKeys STUNReachabilityTestKeys
	if tk["failure"] != nil {
		testKeys.IsAnomaly = true
		failure, ok := tk["failure"].(*string)
		if !ok {
			return testKeys, errors.New("failure key invalid")
		}
		testKeys.Failure = *failure
	}
	return testKeys, nil
}

// LogSummary writes the summary to the standard output
func (n STUNReachability) LogSummary(s string) error {
	return nil
}
