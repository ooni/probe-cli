package im

import (
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/openobservatory/gooni/nettests"
)

// WhatsApp test implementation
type WhatsApp struct {
}

// Run starts the test
func (h WhatsApp) Run(ctl *nettests.Controller) error {
	mknt := mk.NewNettest("Whatsapp")
	ctl.Init(mknt)
	return mknt.Run()
}

// WhatsAppSummary for the test
type WhatsAppSummary struct {
	RegistrationServerBlocking bool
	WebBlocking                bool
	EndpointsBlocking          bool
	Blocked                    bool
}

// Summary generates a summary for a test run
func (h WhatsApp) Summary(tk map[string]interface{}) interface{} {
	var (
		webBlocking          bool
		registrationBlocking bool
		endpointsBlocking    bool
	)

	var computeBlocking = func(key string) bool {
		const blk = "blocked"
		if tk[key] == nil {
			return false
		}
		if tk[key].(string) == blk {
			return true
		}
		return false
	}
	registrationBlocking = computeBlocking("registration_server_status")
	webBlocking = computeBlocking("whatsapp_web_status")
	endpointsBlocking = computeBlocking("whatsapp_endpoints_status")

	return WhatsAppSummary{
		RegistrationServerBlocking: registrationBlocking,
		WebBlocking:                webBlocking,
		EndpointsBlocking:          endpointsBlocking,
		Blocked:                    registrationBlocking || webBlocking || endpointsBlocking,
	}
}

// LogSummary writes the summary to the standard output
func (h WhatsApp) LogSummary(s string) error {
	return nil
}
