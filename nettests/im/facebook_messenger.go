package im

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/openobservatory/gooni/nettests"
)

// FacebookMessenger test implementation
type FacebookMessenger struct {
}

// Run starts the test
func (h FacebookMessenger) Run(ctl *nettests.Controller) error {
	mknt := mk.NewNettest("FacebookMessenger")
	ctl.Init(mknt)
	return mknt.Run()
}

// FacebookMessengerSummary for the test
type FacebookMessengerSummary struct {
	DNSBlocking bool
	TCPBlocking bool
	Blocked     bool
}

// Summary generates a summary for a test run
func (h FacebookMessenger) Summary(tk map[string]interface{}) interface{} {
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

	return FacebookMessengerSummary{
		DNSBlocking: dnsBlocking,
		TCPBlocking: tcpBlocking,
		Blocked:     dnsBlocking || tcpBlocking,
	}
}

// LogSummary writes the summary to the standard output
func (h FacebookMessenger) LogSummary(s string) error {
	return nil
}
