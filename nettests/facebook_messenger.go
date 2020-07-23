package nettests

// FacebookMessenger test implementation
type FacebookMessenger struct {
}

// Run starts the test
func (h FacebookMessenger) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder(
		"facebook_messenger",
	)
	if err != nil {
		return err
	}
	return ctl.Run(builder, []string{""})
}

// FacebookMessengerTestKeys for the test
type FacebookMessengerTestKeys struct {
	DNSBlocking bool `json:"facebook_dns_blocking"`
	TCPBlocking bool `json:"facebook_tcp_blocking"`
	IsAnomaly   bool `json:"-"`
}

// GetTestKeys generates a summary for a test run
func (h FacebookMessenger) GetTestKeys(tk map[string]interface{}) (interface{}, error) {
	var (
		dnsBlocking bool
		tcpBlocking bool
	)
	if tk["facebook_dns_blocking"] == nil {
		dnsBlocking = false
	} else {
		dnsBlocking = tk["facebook_dns_blocking"].(bool)
	}

	if tk["facebook_tcp_blocking"] == nil {
		tcpBlocking = false
	} else {
		tcpBlocking = tk["facebook_tcp_blocking"].(bool)
	}

	return FacebookMessengerTestKeys{
		DNSBlocking: dnsBlocking,
		TCPBlocking: tcpBlocking,
		IsAnomaly:   dnsBlocking || tcpBlocking,
	}, nil
}

// LogSummary writes the summary to the standard output
func (h FacebookMessenger) LogSummary(s string) error {
	return nil
}
